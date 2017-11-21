package extractor

import (
	"compress/gzip"
	"io"
	"os"
)

type tgzExtractor struct{}

func NewTgz() Extractor {
	return &tgzExtractor{}
}

func (e *tgzExtractor) Extract(src *os.File, tarDest io.Writer) error {
	gReader, err := gzip.NewReader(src)
	if err != nil {
		return err
	}

	defer gReader.Close()

	_, err = io.Copy(tarDest, gReader)
	return err
}
