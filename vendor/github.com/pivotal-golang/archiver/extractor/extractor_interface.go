package extractor

import (
	"io"
	"os"
)

type Extractor interface {
	Extract(file *os.File, tarDest io.Writer) error
}
