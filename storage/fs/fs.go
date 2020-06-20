package fs

import (
	"io/ioutil"
	"os"
	"path"
)

type FileSystemStorage struct {
	dir string
}

func New(dir string) *FileSystemStorage {
	fs := new(FileSystemStorage)
	fs.dir = dir
	return fs
}

func (fs *FileSystemStorage) PutObject(objectName string, data []byte) (string, error) {
	fullPath := path.Join(fs.dir, objectName)

	os.MkdirAll(path.Dir(fullPath), os.ModePerm)
	dst, err := os.Create(fullPath)
	if err != nil {
		return "", err
	}
	defer dst.Close()

	_, err = dst.Write(data)
	if err != nil {
		return "", err
	}

	return fullPath, nil
}

func (fs *FileSystemStorage) GetObject(objectName string) ([]byte, error) {
	fullPath := path.Join(fs.dir, objectName)

	data, err := ioutil.ReadFile(fullPath)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (fs *FileSystemStorage) IsExists(objectName string) bool {
	fullPath := path.Join(fs.dir, objectName)

	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return false
	}

	return true
}
