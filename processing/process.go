package processing

import (
	"errors"
	"fmt"
	"github.com/vas3k/pepic/config"
	"github.com/vas3k/pepic/storage"
	"log"
	"mime"
	"path"
	"strconv"
)

func UploadFile(filename string, data []byte) (*ProcessedFile, error) {
	file := &ProcessedFile{
		Filename: filename,
		Mime:     detectMimeType(filename, data),
		Data:     data,
	}

	if file.IsImage() {
		log.Printf("Processing image: %s", file.Mime)
		err := calculateHashName(file)
		if err != nil {
			return file, err
		}

		if !config.App.Images.StoreOriginals {
			err := resizeImage(file, config.App.Images.OriginalLength)
			if err != nil {
				return file, err
			}

			if config.App.Images.AutoConvert != "false" {
				err := convertImage(file, config.App.Images.AutoConvert)
				if err != nil {
					return file, err
				}
			}
		}

		err = storeFile(file, "orig")
		if err != nil {
			return file, err
		}

		return file, nil
	}

	if file.IsVideo() {
		log.Printf("Processing video: %s", file.Mime)
		err := calculateHashName(file)
		if err != nil {
			return file, err
		}

		if !config.App.Videos.StoreOriginals {
			err := transcodeVideo(file, config.App.Videos.OriginalLength)
			if err != nil {
				return file, err
			}

			if config.App.Videos.AutoConvert != "false" {
				err := convertVideo(file, config.App.Videos.AutoConvert)
				if err != nil {
					return file, err
				}
			}
		}

		err = storeFile(file, "orig")
		if err != nil {
			return file, err
		}

		return file, nil
	}

	return nil, errors.New(fmt.Sprintf("unsupported file type: %s", file.Mime))
}

func GetFile(directory string, filename string) (*ProcessedFile, error) {
	canonicalFilename := canonizeFileName(filename)
	filePath := path.Join(directory, canonicalFilename)
	file := &ProcessedFile{
		Filename: filename,
		Path:     filePath,
		Mime:     mime.TypeByExtension(path.Ext(canonicalFilename)),
	}

	if !storage.Main.IsExists(filePath) {
		log.Printf("File does not exists %s", filePath)
		return file, errors.New("file does not exists")
	}

	return file, nil
}

func ResizeFile(filename string, length int) (*ProcessedFile, error) {
	resizePath := path.Join("resize", strconv.Itoa(length))
	file, err := GetFile(resizePath, filename)
	if err == nil {
		// resized file already exists, just return it
		return file, nil
	}

	if file.IsImage() {
		if config.App.Images.LiveResize {
			err := retrieveFileData(file, "orig")
			if err != nil {
				return file, err
			}

			err = resizeImage(file, length)
			if err != nil {
				return file, err
			}

			err = storeFile(file, "orig")
			if err != nil {
				return file, err
			}
		}
		return file, nil
	}

	if file.IsVideo() {
		if config.App.Videos.LiveResize {
			err := retrieveFileData(file, "orig")
			if err != nil {
				return file, err
			}

			err = transcodeVideo(file, length)
			if err != nil {
				return file, err
			}

			err = storeFile(file, "orig")
			if err != nil {
				return file, err
			}
		}
		return file, nil
	}

	return nil, errors.New("file does not exist")
}
