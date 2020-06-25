package processing

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/h2non/bimg"
	"github.com/vas3k/pepic/config"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	_ "image/png"
	"log"
	"strings"
)

func isImage(file *ProcessedFile) bool {
	return strings.HasPrefix(file.Mime, "image/")
}

func mimeTypeToImageType(mimeType string) (bimg.ImageType, error) {
	mapping := map[string]bimg.ImageType{
		"image/jpeg": bimg.JPEG,
		"image/pjpeg": bimg.JPEG,
		"image/webp": bimg.WEBP,
		"image/png": bimg.PNG,
		"image/tiff": bimg.TIFF,
		"image/gif": bimg.GIF,
		"image/svg": bimg.SVG,
		"image/heic": bimg.HEIF,
		"image/heif": bimg.HEIF,
	}
	if imageType, ok := mapping[mimeType]; ok {
		return imageType, nil
	} else {
		return bimg.UNKNOWN, errors.New(fmt.Sprintf("'%s' is not supported", mimeType))
	}
}

func resizeImage(file *ProcessedFile, maxLength int) error {
	log.Printf("Resizing image '%s' to %d px", file.Filename, maxLength)
	if file.Data == nil {
		return errors.New("file data is empty, try reading it first")
	}

	img := bimg.NewImage(file.Data)
	origSize, err := img.Size()
	if err != nil {
		return err
	}

	width, height := fitSize(origSize.Width, origSize.Height, maxLength)
	resizedImg, err := img.Process(bimg.Options{
		Width:  width,
		Height: height,
		Embed:  true,
		StripMetadata: true,
		Quality: config.App.Images.JPEGQuality,
		Compression: config.App.Images.PNGCompression,
	})
	if err != nil {
		return err
	}

	file.Data = resizedImg

	return nil
}

func convertImage(file *ProcessedFile, newMimeType string) error {
	log.Printf("Converting image '%s' to %s", file.Filename, newMimeType)
	if file.Data == nil {
		return errors.New("file data is empty, try reading it first")
	}

	newImgType, err := mimeTypeToImageType(newMimeType)
	if err != nil {
		return err
	}

	img := bimg.NewImage(file.Data)

	// fix PNG -> JPG transparency if needed
	if bimg.DetermineImageType(file.Data) == bimg.PNG && newImgType == bimg.JPEG {
		img = fixPNGTransparency(img)
	}

	convertedImg, err := img.Process(bimg.Options{
		Type: newImgType,
		StripMetadata: true,
		Quality: config.App.Images.JPEGQuality,
		Compression: config.App.Images.PNGCompression,
	})

	newExt, _ := extensionByMimeType(newMimeType)
	file.Data = convertedImg
	file.Mime = newMimeType
	file.Filename = replaceExt(file.Filename, newExt)
	if file.Path != "" {
		file.Path = replaceExt(file.Path, newExt)
	}

	return nil
}

func fixPNGTransparency(img *bimg.Image) *bimg.Image {
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
