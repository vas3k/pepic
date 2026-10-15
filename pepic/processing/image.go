package processing

import (
	"errors"
	"math"
	"sync"

	"github.com/davidbyttow/govips/v2/vips"
	"github.com/vas3k/pepic/pepic/config"
	"github.com/vas3k/pepic/pepic/entity"
	"github.com/vas3k/pepic/pepic/utils"
)

type ImageBackend interface {
	Process(file *entity.ProcessingFile, maxLength int, mimeType string) error
	Resize(file *entity.ProcessingFile, maxLength int) error
}

type imageBackend struct{}

var startImages = sync.OnceValue(func() error {
	return vips.Startup(&vips.Config{
		ConcurrencyLevel: config.App.Safety.ImageThreads,
		MaxCacheMem:      config.App.Safety.VipsCacheBytes,
		MaxCacheSize:     config.App.Safety.VipsCacheEntries,
	})
})

func NewImageBackend() ImageBackend {
	return &imageBackend{}
}

func loadImage(data []byte) (*vips.ImageRef, error) {
	if err := startImages(); err != nil {
		return nil, err
	}
	image, err := vips.NewImageFromBuffer(data)
	if err != nil {
		return nil, err
	}
	if !validDimensions(image.Width(), image.Height(), config.App.Safety.MaxImagePixels) {
		image.Close()
		return nil, ErrInvalidMedia
	}
	return image, nil
}

func (i *imageBackend) Resize(file *entity.ProcessingFile, maxLength int) error {
	return i.Process(file, maxLength, file.Mime)
}

func (i *imageBackend) Process(file *entity.ProcessingFile, maxLength int, mimeType string) error {
	if maxLength <= 0 || maxLength > config.App.Safety.MaxDimension {
		return errors.New("invalid resize dimensions")
	}
	if mimeType == "" || mimeType == "false" {
		mimeType = file.Mime
	}
	canonicalMIME, format, ok := config.App.Media.Format(mimeType)
	if !ok || format.Encode == nil {
		return ErrInvalidMedia
	}
	image, err := loadImage(file.Bytes)
	if err != nil {
		return err
	}
	defer image.Close()
	if err := image.AutoRotate(); err != nil {
		return err
	}
	scale := math.Min(1, float64(maxLength)/float64(max(image.Width(), image.Height())))
	if scale < 1 {
		if err := image.Resize(scale, vips.KernelLanczos3); err != nil {
			return err
		}
	}
	if format.FlattenAlpha && image.HasAlpha() {
		if err := image.Flatten(&vips.Color{R: 255, G: 255, B: 255}); err != nil {
			return err
		}
	}
	if err := image.RemoveMetadata(); err != nil {
		return err
	}
	data, _, err := format.Encode(image, config.ImageEncoding{Quality: config.App.Images.JPEGQuality, Compression: config.App.Images.PNGCompression})
	if err != nil {
		return err
	}
	extension, err := utils.ExtensionByMimeType(canonicalMIME)
	if err != nil {
		return err
	}
	file.Bytes, file.Mime = data, canonicalMIME
	file.Filename = utils.ReplaceExt(file.Filename, extension)
	if file.Path != "" {
		file.Path = utils.ReplaceExt(file.Path, extension)
	}
	return nil
}
