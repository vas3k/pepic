package processing

import (
	"github.com/vas3k/pepic/config"
	"github.com/vas3k/pepic/storage"
)

type ProcessedFile struct {
	Filename string
	Mime     string
	Path     string
	Data     []byte
}

func (p *ProcessedFile) Url() string {
	return config.App.Global.BaseUrl + p.Filename
}

func (p *ProcessedFile) IsImage() bool {
	return isImage(p.Mime)
}

func (p *ProcessedFile) IsVideo() bool {
	return isVideo(p.Mime)
}

func (p *ProcessedFile) Size() int64 {
	return storage.Main.Size(p.Path)
}
