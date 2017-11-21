package builder

import (
	"archive/tar"
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"html/template"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/SUSE/fissile/docker"
	"github.com/SUSE/fissile/model"
	"github.com/SUSE/fissile/scripts/dockerfiles"
	"github.com/SUSE/fissile/util"
	"github.com/SUSE/termui"
)

// PackagesImageBuilder represents a builder of the shared packages layer docker image
type PackagesImageBuilder struct {
	repository           string
	stemcellImageID      string
	stemcellImageName    string
	compiledPackagesPath string
	targetPath           string
	fissileVersion       string
	ui                   *termui.UI
}

// baseImageOverride is used for tests; if not set, we use the correct one
var baseImageOverride string

// NewPackagesImageBuilder creates a new PackagesImageBuilder
func NewPackagesImageBuilder(repository, stemcellImageName, stemcellImageID, compiledPackagesPath, targetPath, fissileVersion string, ui *termui.UI) (*PackagesImageBuilder, error) {
	if err := os.MkdirAll(targetPath, 0755); err != nil {
		return nil, err
	}

	if stemcellImageID == "" {
		imageManager, err := docker.NewImageManager()
		if err != nil {
			return nil, err
		}

		stemcellImage, err := imageManager.FindImage(stemcellImageName)
		if err != nil {
			return nil, err
		}

		stemcellImageID = stemcellImage.ID
	}

	return &PackagesImageBuilder{
		repository:           repository,
		stemcellImageID:      stemcellImageID,
		stemcellImageName:    stemcellImageName,
		compiledPackagesPath: compiledPackagesPath,
		targetPath:           targetPath,
		fissileVersion:       fissileVersion,
		ui:                   ui,
	}, nil
}

// tarWalker is a helper to copy files into a tar stream
type tarWalker struct {
	stream *tar.Writer // The stream to copy the files into
	root   string      // The base directory on disk where the walking started
	prefix string      // The prefix in the tar file the names should have
}

func (w *tarWalker) walk(path string, info os.FileInfo, err error) error {
	if err != nil {
		return err
	}

	header, err := tar.FileInfoHeader(info, "")
	if err != nil {
		return err
	}

	if (info.Mode() & os.ModeSymlink) != 0 {
		linkname, err := os.Readlink(path)
		if err != nil {
			return err
		}
		header.Linkname = linkname
	}

	relPath, err := filepath.Rel(w.root, path)
	if err != nil {
		return err
	}

	header.Name = filepath.Join(w.prefix, relPath)
	if err := w.stream.WriteHeader(header); err != nil {
		return err
	}

	if !info.Mode().IsRegular() {
		return nil
	}

	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.CopyN(w.stream, file, info.Size())
	return err
}

func (p *PackagesImageBuilder) fissileVersionLabel() string {
	return fmt.Sprintf("version.generator.fissile=%s",
		strings.Replace(p.fissileVersion, "+", "_", -1))
}

// determinePackagesLayerBaseImage finds the best base image to use for the
// packages layer image.  Given a list of packages, it returns the base image
// name to use, as well as the set of packages that still need to be inserted.
func (p *PackagesImageBuilder) determinePackagesLayerBaseImage(packages model.Packages) (string, model.Packages, error) {
	baseImageName := p.stemcellImageName
	if baseImageOverride != "" {
		baseImageName = baseImageOverride
	}

	var labels []string
	remainingPackages := make(map[string]*model.Package, len(packages))
	for _, pkg := range packages {
		labels = append(labels, fmt.Sprintf("fingerprint.%s", pkg.Fingerprint))
		remainingPackages[pkg.Fingerprint] = pkg
	}

	var mandatoryLabels = []string{
		p.fissileVersionLabel(),
	}

	dockerManger, err := docker.NewImageManager()
	if err != nil {
		return "", nil, err
	}
	matchedImage, foundLabels, err := dockerManger.FindBestImageWithLabels(baseImageName,
		labels, mandatoryLabels)
	if err != nil {
		return "", nil, err
	}

	// Find the list of packages remaining
	for label := range foundLabels {
		parts := strings.Split(label, ".")
		if len(parts) != 2 || parts[0] != "fingerprint" {
			// Will reach this for mandatory matched labels, i.e. fissile version
			continue
		}
		delete(remainingPackages, parts[1])
	}

	packages = make(model.Packages, 0, len(remainingPackages))
	for _, pkg := range remainingPackages {
		packages = append(packages, pkg)
	}

	return matchedImage, packages, nil
}

// NewDockerPopulator returns a function which can populate a tar stream with the docker context to build the packages layer image with
func (p *PackagesImageBuilder) NewDockerPopulator(roles model.Roles, forceBuildAll bool) func(*tar.Writer) error {
	return func(tarWriter *tar.Writer) error {
		var err error
		if len(roles) == 0 {
			return fmt.Errorf("No roles to build")
		}

		// Collect compiled packages
		foundFingerprints := make(map[string]struct{})
		var packages model.Packages
		for _, role := range roles {
			for _, roleJob := range role.RoleJobs {
				for _, pkg := range roleJob.Packages {
					if _, ok := foundFingerprints[pkg.Fingerprint]; ok {
						// Package has already been found (possibly due to a different role)
						continue
					}
					packages = append(packages, pkg)
					foundFingerprints[pkg.Fingerprint] = struct{}{}
				}
			}
		}

		// Generate dockerfile
		dockerfile := bytes.Buffer{}
		baseImageName := p.stemcellImageName
		if !forceBuildAll {
			baseImageName, packages, err = p.determinePackagesLayerBaseImage(packages)
			if err != nil {
				return err
			}
		}
		if err = p.generateDockerfile(baseImageName, packages, &dockerfile); err != nil {
			return err
		}
		err = util.WriteToTarStream(tarWriter, dockerfile.Bytes(), tar.Header{
			Name: "Dockerfile",
		})
		if err != nil {
			return err
		}

		// Make sure we have the directory, even if we have no packages to add
		err = util.WriteToTarStream(tarWriter, []byte{}, tar.Header{
			Name:     "packages-src",
			Mode:     0755,
			Typeflag: tar.TypeDir,
		})
		if err != nil {
			return err
		}

		// Actually insert the packages into the tar stream
		for _, pkg := range packages {
			walker := &tarWalker{
				stream: tarWriter,
				root:   pkg.GetPackageCompiledDir(p.compiledPackagesPath),
				prefix: filepath.Join("packages-src", pkg.Fingerprint),
			}
			if err = filepath.Walk(walker.root, walker.walk); err != nil {
				return err
			}
		}

		return nil
	}
}

// generateDockerfile builds a docker file for the shared packages layer.
func (p *PackagesImageBuilder) generateDockerfile(baseImage string, packages model.Packages, outputFile io.Writer) error {
	context := map[string]interface{}{
		"base_image":      baseImage,
		"packages":        packages,
		"fissile_version": p.fissileVersionLabel(),
	}
	asset, err := dockerfiles.Asset("Dockerfile-packages")
	if err != nil {
		return err
	}

	dockerfileTemplate := template.New("Dockerfile")
	dockerfileTemplate, err = dockerfileTemplate.Parse(string(asset))
	if err != nil {
		return err
	}

	return dockerfileTemplate.Execute(outputFile, context)
}

// GetPackagesLayerImageName generates a docker image name for the amalgamation holding all packages used in the specified roles
func (p *PackagesImageBuilder) GetPackagesLayerImageName(roleManifest *model.RoleManifest, roles model.Roles) (string, error) {
	// Get the list of packages; use the fingerprint to ensure we have no repeats
	pkgMap := make(map[string]*model.Package)
	for _, r := range roles {
		for _, j := range r.RoleJobs {
			for _, pkg := range j.Packages {
				pkgMap[pkg.Fingerprint] = pkg
			}
		}
	}

	// Sort the packages to have a consistent order
	pkgs := make(model.Packages, 0, len(pkgMap))
	for _, pkg := range pkgMap {
		pkgs = append(pkgs, pkg)
	}
	sort.Sort(pkgs)

	// Get the hash
	hasher := sha1.New()
	hasher.Write([]byte(fmt.Sprintf("%s:%s", p.fissileVersion, p.stemcellImageID)))
	for _, pkg := range pkgs {
		hasher.Write([]byte(strings.Join([]string{"", pkg.Fingerprint, pkg.Name, pkg.SHA1}, "\000")))
	}

	imageName := util.SanitizeDockerName(fmt.Sprintf("%s-role-packages", p.repository))
	imageTag := util.SanitizeDockerName(hex.EncodeToString(hasher.Sum(nil)))
	return fmt.Sprintf("%s:%s", imageName, imageTag), nil
}
