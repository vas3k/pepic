package processing

import (
	"crypto/sha256"
	"encoding/hex"
	"github.com/vas3k/pepic/config"
	"github.com/vas3k/pepic/storage"
	"log"
	"path"
	"strings"
)

func calculateHashName(file *ProcessedFile) error {
	log.Printf("Calculating file name: %s", file.Filename)
	ext, err := extensionByMimeType(file.Mime)
	if err != nil {
		return err
	}

	sum := sha256.Sum256(file.Data)
	file.Filename = strings.ToLower(hex.EncodeToString(sum[:]) + ext)

	return nil
}

func canonizeFileName(filename string) string {
	return splitFileNameNGrams(filename, config.App.Global.FileTreeSplitChars, 10)
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
