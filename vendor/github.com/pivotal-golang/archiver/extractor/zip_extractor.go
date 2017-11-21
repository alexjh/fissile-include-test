package extractor

import (
	"archive/tar"
	"archive/zip"
	"io"
	"os"
)

type zipExtractor struct{}

func NewZip() Extractor {
	return &zipExtractor{}
}

func (e *zipExtractor) Extract(zipFile *os.File, tarDest io.Writer) error {
	zipReader, err := zip.OpenReader(zipFile.Name())
	if err != nil {
		return err
	}

	defer zipReader.Close()

	tarWriter := tar.NewWriter(tarDest)

	for _, zipEntry := range zipReader.File {
		err := writeZipEntryToTar(tarWriter, zipEntry)
		if err != nil {
			return err
		}
	}

	return nil
}

func writeZipEntryToTar(tarWriter *tar.Writer, zipEntry *zip.File) error {
	zipInfo := zipEntry.FileInfo()

	tarHeader, err := tar.FileInfoHeader(zipInfo, "")
	if err != nil {
		return err
	}

	// file info only populates the base name; we want the full path
	tarHeader.Name = zipEntry.FileHeader.Name

	zipReader, err := zipEntry.Open()
	if err != nil {
		return err
	}

	defer zipReader.Close()

	err = tarWriter.WriteHeader(tarHeader)
	if err != nil {
		return err
	}

	_, err = io.Copy(tarWriter, zipReader)
	if err != nil {
		return err
	}

	err = tarWriter.Flush()
	if err != nil {
		return err
	}

	return nil
}
