package utils

import (
	"mime"
	"net/http"
	"path"
)

func DetectMimeType(filename string, data []byte) string {
	mimeType := http.DetectContentType(data)
	if mimeType == "application/octet-stream" {
		mimeType = mime.TypeByExtension(path.Ext(filename))
	}
	return mimeType
}

func FitSize(origWidth int, origHeight int, length int) (int, int) {
	if origWidth > origHeight {
		return length, (origHeight * length) / origWidth
	} else if origWidth < origHeight {
		return (origWidth * length) / origHeight, length
	} else {
		return length, length
	}
}
