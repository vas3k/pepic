package processing

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/vas3k/pepic/config"
	"image"
	"image/color"
	"image/png"
	"log"
	"mime"

	"github.com/disintegration/imaging"
)

func isImage(mimeType string) bool {
	return mimeType == "image/jpeg" || mimeType == "image/png" || mimeType == "image/gif"
}

func resizeImage(file *ProcessedFile, maxLength int) error {
	log.Printf("Resizing image '%s' to %d px", file.Filename, maxLength)
	if file.Data == nil {
		return errors.New("file data is empty, try reading it first")
	}

	image, err := imaging.Decode(bytes.NewReader(file.Data), imaging.AutoOrientation(true))
	if err != nil {
		return err
	}

	imageFormat, err := imaging.FormatFromFilename(file.Filename)
	if err != nil {
		return err
	}

	resizedImage := imaging.Fit(image, maxLength, maxLength, imaging.Lanczos)

	bytesBuffer := new(bytes.Buffer)
	err = imaging.Encode(
		bytesBuffer,
		resizedImage,
		imageFormat,
		imaging.JPEGQuality(config.App.Images.JPEGQuality),
		imaging.PNGCompressionLevel(png.CompressionLevel(config.App.Images.PNGCompression)),
	)
	if err != nil {
		return err
	}

	file.Data = bytesBuffer.Bytes()

	return nil
}

func convertImage(file *ProcessedFile, newMimeType string) error {
	log.Printf("Converting image '%s' to %s", file.Filename, newMimeType)
	if file.Data == nil {
		return errors.New("file data is empty, try reading it first")
	}

	if !isImage(newMimeType) {
		return errors.New(fmt.Sprintf("'%s' is not an image mime type", newMimeType))
	}

	image, err := imaging.Decode(bytes.NewReader(file.Data))
	if err != nil {
		return err
	}

	ext, _ := mime.ExtensionsByType(newMimeType)
	newExt := ext[0]
	imageFormat, err := imaging.FormatFromExtension(newExt)
	if err != nil {
		return err
	}

	// fix PNG transparency if needed
	if image.ColorModel() == color.RGBAModel && newMimeType == "image/jpeg" {
		fixBlackPNGBackground(&image)
	}

	// encode the result back to bytes
	bytesBuffer := new(bytes.Buffer)
	err = imaging.Encode(
		bytesBuffer,
		image,
		imageFormat,
		imaging.JPEGQuality(config.App.Images.JPEGQuality),
		imaging.PNGCompressionLevel(png.CompressionLevel(config.App.Images.PNGCompression)),
	)
	if err != nil {
		return err
	}

	file.Data = bytesBuffer.Bytes()
	file.Mime = newMimeType
	file.Filename = replaceExt(file.Filename, newExt)
	if file.Path != "" {
		file.Path = replaceExt(file.Path, newExt)
	}

	return nil
}

func fixBlackPNGBackground(image *image.Image) error {
	// TODO: this
	return nil
}
