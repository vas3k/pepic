package storage

import (
	"crypto/rand"
	"errors"
	"io"
	"net/http"
	"os"
	"path"

	"github.com/labstack/echo/v5"
	"github.com/vas3k/pepic/pepic/config"
)

type FileSystemBackend struct {
	dir string
}

func NewFileSystemBackend(dir string) *FileSystemBackend {
	return &FileSystemBackend{dir: dir}
}

func (fs *FileSystemBackend) PutObject(objectName string, data []byte) (string, error) {
	if err := os.MkdirAll(fs.dir, 0755); err != nil {
		return "", err
	}
	root, err := os.OpenRoot(fs.dir)
	if err != nil {
		return "", err
	}
	defer root.Close()
	if err := root.MkdirAll(path.Dir(objectName), 0755); err != nil {
		return "", err
	}
	temporary := path.Join(path.Dir(objectName), ".pepic-"+rand.Text())
	file, err := root.OpenFile(temporary, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return "", err
	}
	defer file.Close()
	defer root.Remove(temporary)
	if _, err := file.Write(data); err != nil {
		return "", err
	}
	if err := file.Close(); err != nil {
		return "", err
	}
	if err := root.Rename(temporary, objectName); err != nil {
		return "", err
	}
	return path.Join(fs.dir, objectName), nil
}

func (fs *FileSystemBackend) open(objectName string) (*os.File, error) {
	root, err := os.OpenRoot(fs.dir)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	return root.Open(objectName)
}

func (fs *FileSystemBackend) GetObject(objectName string) ([]byte, error) {
	file, err := fs.open(objectName)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	limit, err := config.UploadLimit()
	if err != nil {
		return nil, err
	}
	data, err := io.ReadAll(io.LimitReader(file, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, errors.New("stored file exceeds upload limit")
	}
	return data, nil
}

func (fs *FileSystemBackend) IsExists(objectName string) bool {
	file, err := fs.open(objectName)
	if err != nil {
		return false
	}
	defer file.Close()
	info, err := file.Stat()
	return err == nil && info.Mode().IsRegular()
}

func (fs *FileSystemBackend) Size(objectName string) int64 {
	file, err := fs.open(objectName)
	if err != nil {
		return 0
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return 0
	}
	return info.Size()
}

func (fs *FileSystemBackend) Proxy(c *echo.Context, objectName string) error {
	mimeType, _, allowed := config.App.Media.FormatByExtension(path.Ext(objectName))
	if !allowed {
		return echo.ErrNotFound
	}
	file, err := fs.open(objectName)
	if err != nil {
		return echo.ErrNotFound
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return echo.ErrNotFound
	}
	headers := c.Response().Header()
	headers.Set("Content-Type", mimeType)
	headers.Set("X-Content-Type-Options", "nosniff")
	headers.Set("Content-Security-Policy", "default-src 'none'; sandbox")
	http.ServeContent(c.Response(), c.Request(), info.Name(), info.ModTime(), file)
	return nil
}
