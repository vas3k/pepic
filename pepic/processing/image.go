package processing

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/h2non/bimg"
	"github.com/vas3k/pepic/pepic/config"
	"github.com/vas3k/pepic/pepic/entity"
	"github.com/vas3k/pepic/pepic/utils"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	_ "image/png"
	"log"
)

type ImageBackend interface {
	AutoRotate(file *entity.ProcessingFile) error
	Resize(file *entity.ProcessingFile, maxLength int) error
	Convert(file *entity.ProcessingFile, newMimeType string) error
}

type imageBackend struct {
}

func NewImageBackend() ImageBackend {
	return &imageBackend{}
}

func (i *imageBackend) AutoRotate(file *entity.ProcessingFile) error {
	log.Printf("Auto-rotate image '%s'", file.Filename)
	if file.Bytes == nil {
		return errors.New("file data is empty, try reading it first")
	}

	img := bimg.NewImage(file.Bytes)
	rotatedImg, err := img.AutoRotate()
	if err != nil {
		return err
	}

	file.Bytes = rotatedImg

	return nil
}

func (i *imageBackend) Resize(file *entity.ProcessingFile, maxLength int) error {
	log.Printf("Resizing image '%s' to %d px", file.Filename, maxLength)
	if file.Bytes == nil {
		return errors.New("file data is empty, try reading it first")
	}

	img := bimg.NewImage(file.Bytes)
	origSize, err := img.Size()
	if err != nil {
		return err
	}

	width, height := utils.FitSize(origSize.Width, origSize.Height, maxLength)
	log.Printf("Orig width '%d' height %d px", origSize.Width, origSize.Height)
	resizedImg, err := img.Process(bimg.Options{
		Width:         width,
		Height:        height,
		Embed:         true,
		StripMetadata: true,
		Quality:       config.App.Images.JPEGQuality,
		Compression:   config.App.Images.PNGCompression,
	})
	if err != nil {
		return err
	}

	file.Bytes = resizedImg

	return nil
}

func (i *imageBackend) Convert(file *entity.ProcessingFile, newMimeType string) error {
	log.Printf("Converting image '%s' to %s", file.Filename, newMimeType)
	if file.Bytes == nil {
		return errors.New("file data is empty, try reading it first")
	}

	newImgType, err := i.mimeTypeToImageType(newMimeType)
	if err != nil {
		return err
	}

	img := bimg.NewImage(file.Bytes)

	// fix PNG -> JPG transparency if needed
	if bimg.DetermineImageType(file.Bytes) == bimg.PNG && newImgType == bimg.JPEG {
		img = i.fixPNGTransparency(img)
	}

	convertedImg, err := img.Process(bimg.Options{
		Type:          newImgType,
		StripMetadata: true,
		Quality:       config.App.Images.JPEGQuality,
		Compression:   config.App.Images.PNGCompression,
	})

	newExt, _ := utils.ExtensionByMimeType(newMimeType)
	file.Bytes = convertedImg
	file.Mime = newMimeType
	file.Filename = utils.ReplaceExt(file.Filename, newExt)
	if file.Path != "" {
		file.Path = utils.ReplaceExt(file.Path, newExt)
	}

	return nil
}

func (i imageBackend) fixPNGTransparency(img *bimg.Image) *bimg.Image {
	// convert to go image because bimg has no drawing features
	origImg, _, err := image.Decode(bytes.NewReader(img.Image()))
	if err != nil {
		return img
	}

	// draw white square and paste image on top of it
	newImg := image.NewRGBA(origImg.Bounds())
	draw.Draw(newImg, newImg.Bounds(), &image.Uniform{color.White}, image.Point{}, draw.Src)
	draw.Draw(newImg, newImg.Bounds(), origImg, origImg.Bounds().Min, draw.Over)

	// convert back to bytes
	buf := new(bytes.Buffer)
	err = jpeg.Encode(buf, newImg, nil)
	if err != nil {
		return img
	}
	return bimg.NewImage(buf.Bytes())
}

func (i imageBackend) mimeTypeToImageType(mimeType string) (bimg.ImageType, error) {
	mapping := map[string]bimg.ImageType{
		"image/jpeg":  bimg.JPEG,
		"image/pjpeg": bimg.JPEG,
		"image/webp":  bimg.WEBP,
		"image/png":   bimg.PNG,
		"image/tiff":  bimg.TIFF,
		"image/gif":   bimg.GIF,
		"image/svg":   bimg.SVG,
		"image/heic":  bimg.HEIF,
		"image/heif":  bimg.HEIF,
	}
	if imageType, ok := mapping[mimeType]; ok {
		return imageType, nil
	} else {
		return bimg.UNKNOWN, errors.New(fmt.Sprintf("'%s' is not supported", mimeType))
	}
}
