package storage

import (
	"errors"
	"github.com/labstack/echo/v4"
	"github.com/vas3k/pepic/pepic/entity"
	"github.com/vas3k/pepic/pepic/utils"
	"log"
	"mime"
	"path"
)

type Backend interface {
	PutObject(objectName string, data []byte) (string, error)
	GetObject(objectName string) ([]byte, error)
	Size(objectName string) int64
	IsExists(objectName string) bool
	Proxy(c echo.Context, objectName string) error
}

type Storage interface {
	GetFile(directory string, filename string) (*entity.ProcessingFile, error)
	ReadFileBytes(file *entity.ProcessingFile, directories ...string) error
	StoreFile(file *entity.ProcessingFile, directories ...string) error
	Proxy(c echo.Context, objectName string) error
}

type storage struct {
	Backend Backend
}

func (s *storage) GetFile(directory string, filename string) (*entity.ProcessingFile, error) {
	canonicalFilename := utils.CanonizeFileName(filename)
	filePath := path.Join(directory, canonicalFilename)
	file := &entity.ProcessingFile{
		Filename: filename,
		Path:     filePath,
		Mime:     mime.TypeByExtension(path.Ext(canonicalFilename)),
	}

	if !s.Backend.IsExists(filePath) {
		log.Printf("File does not exists %s", filePath)
		return file, errors.New("file does not exists")
	}

	file.Size = s.Backend.Size(filePath)

	return file, nil
}

func (s *storage) ReadFileBytes(file *entity.ProcessingFile, directories ...string) error {
	log.Printf("Reading file contents: %s", file.Filename)
	canonicalPath := path.Join(
		path.Join(directories...),
		utils.CanonizeFileName(file.Filename),
	)

	data, err := s.Backend.GetObject(canonicalPath)
	if err != nil {
		log.Fatalf("Error getting file '%s' from storage: %s", canonicalPath, err)
		return err
	}

	file.Bytes = data
	file.Size = int64(len(data))

	return nil
}

func (s *storage) StoreFile(file *entity.ProcessingFile, directories ...string) error {
	log.Printf("Storing file data: %s", file.Filename)
	canonicalPath := path.Join(
		path.Join(directories...),
		utils.CanonizeFileName(file.Filename),
	)

	_, err := s.Backend.PutObject(canonicalPath, file.Bytes)
	if err != nil {
		log.Fatalf("Error writing file '%s' to storage: %s", canonicalPath, err)
		return err
	}

	file.Path = canonicalPath

	return nil
}

func (s *storage) Proxy(c echo.Context, objectName string) error {
	return s.Backend.Proxy(c, objectName)
}

func NewStorage(backend Backend) Storage {
	return &storage{
		Backend: backend,
	}
}
