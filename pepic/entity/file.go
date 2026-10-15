package entity

import (
	"github.com/vas3k/pepic/pepic/config"
)

type ProcessingFile struct {
	Filename string
	Mime     string
	Path     string
	Bytes    []byte
	Size     int64
}

func (p *ProcessingFile) Url() string {
	return config.App.Global.BaseUrl + p.Filename
}

func (p *ProcessingFile) IsAnimated() bool {
	_, format, ok := config.App.Media.Format(p.Mime)
	return ok && format.Animated
}

func (p *ProcessingFile) IsImage() bool {
	_, format, ok := config.App.Media.Format(p.Mime)
	return ok && format.Kind == "image"
}

func (p *ProcessingFile) IsVideo() bool {
	_, format, ok := config.App.Media.Format(p.Mime)
	return ok && format.Kind == "video"
}

func (p *ProcessingFile) Processor() string {
	_, format, _ := config.App.Media.Format(p.Mime)
	return format.Processor
}
