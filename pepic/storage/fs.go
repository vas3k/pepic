package storage

import (
	"github.com/labstack/echo/v4"
	"io/ioutil"
	"os"
	"path"
)

type FileSystemBackend struct {
	dir string
}

func NewFileSystemBackend(dir string) *FileSystemBackend {
	fs := new(FileSystemBackend)
	fs.dir = dir
	return fs
}

func (fs *FileSystemBackend) PutObject(objectName string, data []byte) (string, error) {
	fullPath := path.Join(fs.dir, objectName)

	if err := os.MkdirAll(path.Dir(fullPath), os.ModePerm); err != nil {
		return "", err
	}

	dst, err := os.Create(fullPath)

	if err != nil {
		return "", err
	}

	defer dst.Close()

	if _, err = dst.Write(data); err != nil {
		return "", err
	}

	return fullPath, nil
}

func (fs *FileSystemBackend) GetObject(objectName string) ([]byte, error) {
	fullPath := path.Join(fs.dir, objectName)

	data, err := ioutil.ReadFile(fullPath)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (fs *FileSystemBackend) IsExists(objectName string) bool {
	fullPath := path.Join(fs.dir, objectName)

	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return false
	}

	return true
}

func (fs *FileSystemBackend) Size(objectName string) int64 {
	file, err := os.Open(path.Join(fs.dir, objectName))
	defer file.Close()
	if err != nil {
		return 0
	}

	info, err := file.Stat()
	if err != nil {
		return 0
	}
	return info.Size()
}

func (fs *FileSystemBackend) Proxy(c echo.Context, objectName string) error {
	return c.File(path.Join(fs.dir, objectName))
}
