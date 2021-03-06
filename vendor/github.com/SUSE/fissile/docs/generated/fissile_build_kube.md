## fissile build kube

Creates Kubernetes configuration files.

### Synopsis


Creates Kubernetes configuration files.

```
fissile build kube
```

### Options

```
  -D, --defaults-file string   Env files that contain defaults for the parameters generated by kube
      --output-dir string      Kubernetes configuration files will be written to this directory (default ".")
      --use-memory-limits      Include memory limits when generating kube configurations (default true)
```

### Options inherited from parent commands

```
  -c, --cache-dir string             Local BOSH cache directory. (default "~/.bosh/cache")
      --config string                config file (default is $HOME/.fissile.yaml)
  -d, --dark-opinions string         Path to a BOSH deployment manifest file that contains properties that should not have opinionated defaults.
      --docker-organization string   Docker organization used when referencing image names
      --docker-registry string       Docker registry used when referencing image names
  -l, --light-opinions string        Path to a BOSH deployment manifest file that contains properties to be used as defaults.
  -M, --metrics string               Path to a CSV file to store timing metrics into.
  -o, --output string                Choose output format, one of human, json, or yaml (currently only for 'show properties') (default "human")
  -r, --release string               Path to dev BOSH release(s).
  -n, --release-name string          Name of a dev BOSH release; if empty, default configured dev release name will be used
  -v, --release-version string       Version of a dev BOSH release; if empty, the latest dev release will be used
  -p, --repository string            Repository name prefix used to create image names. (default "fissile")
  -m, --role-manifest string         Path to a yaml file that details which jobs are used for each role.
  -V, --verbose                      Enable verbose output.
  -w, --work-dir string              Path to the location of the work directory. (default "/var/fissile")
  -W, --workers int                  Number of workers to use; zero means determine based on CPU count.
```

### SEE ALSO
* [fissile build](fissile_build.md)	 - Has subcommands to build all images and necessary artifacts.

###### Auto generated by spf13/cobra on 12-Nov-2017
