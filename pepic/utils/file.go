package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"path"
	"strings"

	"github.com/vas3k/pepic/pepic/config"
	"github.com/vas3k/pepic/pepic/entity"
)

func ExtensionByMimeType(mimeType string) (string, error) {
	_, format, ok := config.App.Media.Format(mimeType)
	if !ok {
		return "", fmt.Errorf("unsupported media type %q", mimeType)
	}
	canonicalExtensions := config.App.CanonicalExtensions
	if extension, ok := canonicalExtensions[format.Extension]; ok {
		return extension, nil
	}
	return format.Extension, nil
}

func SplitFileNameNGrams(filename string, n int, stop int) string {
	ext := path.Ext(filename)
	base := strings.TrimSuffix(filename, ext)

	var resultBuilder strings.Builder
	resultBuilder.Grow(len(base) + 5)
	for i, r := range base {
		resultBuilder.WriteRune(r)
		if (i+1)%n == 0 && i < len(base)-1 {
			resultBuilder.WriteRune('/')
		}
		if i >= stop {
			resultBuilder.WriteString(base[i+1:])
			break
		}
	}
	return resultBuilder.String() + ext
}

func ReplaceExt(filename string, newExt string) string {
	ext := path.Ext(filename)
	return filename[:len(filename)-len(ext)] + newExt
}

func CalculateHashName(file *entity.ProcessingFile) (string, error) {
	log.Printf("Calculating file name: %s", file.Filename)
	ext, err := ExtensionByMimeType(file.Mime)
	if err != nil {
		return "", err
	}

	sum := sha256.Sum256(file.Bytes)

	return strings.ToLower(hex.EncodeToString(sum[:]) + ext), nil
}

func CanonizeFileName(filename string) string {
	return SplitFileNameNGrams(filename, config.App.Global.FileTreeSplitChars, 10)
}
