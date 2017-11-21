package extractor

import (
	"net/http"
	"os"
)

func mimeType(fd *os.File) (string, error) {
	data := make([]byte, 512)

	_, err := fd.Read(data)
	if err != nil {
		return "", err
	}

	_, err = fd.Seek(0, 0)
	if err != nil {
		return "", err
	}

	return http.DetectContentType(data), nil
}
