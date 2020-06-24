package storage

import (
	"github.com/labstack/echo/v4"
	"github.com/vas3k/pepic/config"
	"github.com/vas3k/pepic/storage/fs"
	"log"
)

type Storage interface {
	PutObject(objectName string, data []byte) (string, error)
	GetObject(objectName string) ([]byte, error)
	Size(objectName string) int64
	IsExists(objectName string) bool
	Proxy(c echo.Context, objectName string) error
}

var Main Storage

func init() {
	switch config.App.Storage.Type {
	case "fs":
		Main = fs.New(config.App.Storage.Dir)
	default:
		log.Fatalf("Storage type '%s' is not supported", config.App.Storage.Type)
	}
}
