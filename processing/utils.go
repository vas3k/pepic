package processing

import (
	"mime"
	"net/http"
	"path"
	"strings"
)

var canonicalExtensions = map[string]string{
	".jpeg": ".jpg",
	".qt":   ".mov",
}

func extensionByMimeType(mimeType string) (string, error) {
	exts, err := mime.ExtensionsByType(mimeType)
	if err != nil {
		return "", err
	}

	var ext string
	if val, ok := canonicalExtensions[exts[0]]; ok {
		ext = val
	} else {
		ext = exts[0]
	}

	return ext, nil
}

func splitFileNameNGrams(filename string, n int, stop int) string {
	ext := path.Ext(filename)
	base := strings.TrimSuffix(filename, ext)

	var resultBuilder strings.Builder
	resultBuilder.Grow(len(base) + 5)
	for i, r := range base {
		resultBuilder.WriteRune(r)
		if (i + 1) % n == 0 && i != len(base) {
			resultBuilder.WriteRune('/')
		}
		if i >= stop {
			resultBuilder.WriteString(base[i+1:])
			break
		}
	}
	return resultBuilder.String() + ext
}

func detectMimeType(filename string, data []byte) string {
	mimeType := http.DetectContentType(data)
	if mimeType == "application/octet-stream" {
		mimeType = mime.TypeByExtension(path.Ext(filename))
	}
	return mimeType
}

func replaceExt(filename string, newExt string) string {
	ext := path.Ext(filename)
	return filename[:len(filename)-len(ext)] + newExt
}
