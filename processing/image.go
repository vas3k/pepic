package processing

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/disintegration/imaging"
	"github.com/vas3k/pepic/config"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"log"
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

	newExt, _ := extensionByMimeType(newMimeType)
	imageFormat, err := imaging.FormatFromExtension(newExt)
	if err != nil {
		return err
	}

	// fix PNG -> JPG transparency if needed
	if file.Mime == "image/png" && newMimeType == "image/jpeg" {
		image = fixPNGTransparency(image)
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

func fixPNGTransparency(img image.Image) image.Image {
	newImg := image.NewRGBA(img.Bounds())
	draw.Draw(newImg, newImg.Bounds(), &image.Uniform{color.White}, image.Point{}, draw.Src)
	draw.Draw(newImg, newImg.Bounds(), img, img.Bounds().Min, draw.Over)
	return image.Image(newImg)
}
