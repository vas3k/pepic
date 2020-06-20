package processing

import (
	"crypto/sha256"
	"encoding/hex"
	"github.com/vas3k/pepic/storage"
	"log"
	"mime"
	"path"
	"strings"
)

type ProcessedFile struct {
	Filename string
	Mime     string
	Path     string
	Data     []byte
}

var canonicalExtensions = map[string]string{
	".jpeg": ".jpg",
	".qt":   ".mov",
}

func calculateHashName(file *ProcessedFile) error {
	log.Printf("Calculating file name: %s", file.Filename)
	exts, err := mime.ExtensionsByType(file.Mime)
	if err != nil {
		return err
	}

	var ext string
	if val, ok := canonicalExtensions[exts[0]]; ok {
		ext = val
	} else {
		ext = exts[0]
	}

	sum := sha256.Sum256(file.Data)
	file.Filename = strings.ToLower(hex.EncodeToString(sum[:]) + ext)

	return nil
}

func canonizeFileName(filename string) string {
	return splitFileNameNGrams(filename, 3, 10)
}

func storeFile(file *ProcessedFile, directories ...string) error {
	log.Printf("Storing file data: %s", file.Filename)
	canonicalPath := path.Join(
		path.Join(directories...),
		canonizeFileName(file.Filename),
	)

	savedPath, err := storage.Main.PutObject(canonicalPath, file.Data)
	if err != nil {
		log.Fatalf("Error writing file '%s' to storage: %s", canonicalPath, err)
		return err
	}

	file.Path = savedPath

	return nil
}

func retrieveFileData(file *ProcessedFile, directories ...string) error {
	log.Printf("Reading file contents: %s", file.Filename)
	canonicalPath := path.Join(
		path.Join(directories...),
		canonizeFileName(file.Filename),
	)

	data, err := storage.Main.GetObject(canonicalPath)
	if err != nil {
		log.Fatalf("Error getting file '%s' from storage: %s", canonicalPath, err)
		return err
	}

	file.Data = data

	return nil
}
