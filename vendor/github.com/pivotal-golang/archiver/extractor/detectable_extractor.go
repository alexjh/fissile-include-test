package extractor

import (
	"fmt"
	"io"
	"os"
)

type detectableExtractor struct{}

func NewDetectable() Extractor {
	return &detectableExtractor{}
}

func (e *detectableExtractor) Extract(src *os.File, dest io.Writer) error {
	srcType, err := mimeType(src)
	if err != nil {
		return err
	}

	switch srcType {
	case "application/zip":
		err := NewZip().Extract(src, dest)
		if err != nil {
			return err
		}
	case "application/x-gzip":
		err := NewTgz().Extract(src, dest)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported archive type: %s", srcType)
	}

	return nil
}
