package config

import "github.com/davidbyttow/govips/v2/vips"

func defaultMedia() MediaConfig {
	return MediaConfig{Formats: map[string]MediaFormat{
		"image/jpeg": {
			Kind:      "image",
			Processor: "vips",
			Extension: ".jpeg",
			Aliases:   []string{"image/pjpeg"},
			ImageType: vips.ImageTypeJPEG,
			Encode: func(image *vips.ImageRef, options ImageEncoding) ([]byte, *vips.ImageMetadata, error) {
				params := vips.NewJpegExportParams()
				params.Quality = options.Quality
				return image.ExportJpeg(params)
			},
			FlattenAlpha: true,
		},
		"image/png": {
			Kind:      "image",
			Processor: "vips",
			Extension: ".png",
			ImageType: vips.ImageTypePNG,
			Encode: func(image *vips.ImageRef, options ImageEncoding) ([]byte, *vips.ImageMetadata, error) {
				params := vips.NewPngExportParams()
				params.Compression = options.Compression
				if legacy, ok := map[int]int{-1: 0, -2: 1, -3: 9}[options.Compression]; ok {
					params.Compression = legacy
				}
				return image.ExportPng(params)
			},
			GoDecode: true,
		},
		"image/webp": {
			Kind:      "image",
			Processor: "vips",
			Extension: ".webp",
			ImageType: vips.ImageTypeWEBP,
			Encode: func(image *vips.ImageRef, options ImageEncoding) ([]byte, *vips.ImageMetadata, error) {
				params := vips.NewWebpExportParams()
				params.Quality = options.Quality
				return image.ExportWebp(params)
			},
		},
		"image/tiff": {
			Kind:      "image",
			Processor: "vips",
			Extension: ".tiff",
			ImageType: vips.ImageTypeTIFF,
			Encode: func(image *vips.ImageRef, options ImageEncoding) ([]byte, *vips.ImageMetadata, error) {
				params := vips.NewTiffExportParams()
				params.Quality = options.Quality
				return image.ExportTiff(params)
			},
		},
		"image/heif": {
			Kind:      "image",
			Processor: "vips",
			Extension: ".heif",
			Aliases:   []string{"image/heic"},
			ImageType: vips.ImageTypeHEIF,
			Encode: func(image *vips.ImageRef, options ImageEncoding) ([]byte, *vips.ImageMetadata, error) {
				params := vips.NewHeifExportParams()
				params.Quality = options.Quality
				return image.ExportHeif(params)
			},
		},
		"image/avif": {
			Kind:      "image",
			Processor: "vips",
			Extension: ".avif",
			ImageType: vips.ImageTypeAVIF,
			Encode: func(image *vips.ImageRef, options ImageEncoding) ([]byte, *vips.ImageMetadata, error) {
				params := vips.NewAvifExportParams()
				params.Quality = options.Quality
				return image.ExportAvif(params)
			},
		},
		"image/gif": {
			Kind:            "image",
			Processor:       "ffmpeg",
			Extension:       ".gif",
			ImageType:       vips.ImageTypeGIF,
			Animated:        true,
			AllowStillImage: true,
			GoDecode:        true,
			Demuxer:         "gif",
		},
		"video/mp4": {
			Kind:        "video",
			Processor:   "ffmpeg",
			Extension:   ".mp4",
			Demuxer:     "mov",
			InputArgs:   []string{"-enable_drefs", "0", "-use_absolute_path", "0"},
			UseMovFlags: true,
		},
		"video/quicktime": {
			Kind:        "video",
			Processor:   "ffmpeg",
			Extension:   ".mov",
			Demuxer:     "mov",
			InputArgs:   []string{"-enable_drefs", "0", "-use_absolute_path", "0"},
			UseMovFlags: true,
			Signatures:  []Signature{{Offset: 4, Hex: "6674797071742020"}},
		},
		"video/webm": {
			Kind:      "video",
			Processor: "ffmpeg",
			Extension: ".webm",
			Demuxer:   "matroska",
		},
		"video/x-msvideo": {
			Kind:      "video",
			Processor: "ffmpeg",
			Extension: ".avi",
			Aliases:   []string{"video/avi"},
			Demuxer:   "avi",
		},
		"video/ogg": {
			Kind:      "video",
			Processor: "ffmpeg",
			Extension: ".ogv",
			Aliases:   []string{"application/ogg"},
			Demuxer:   "ogg",
		},
	}}
}

func defaultCanonicalExtensions() map[string]string {
	return map[string]string{
		".jpeg": ".jpg", ".jpe": ".jpg", ".jfif": ".jpg",
		".qt": ".mov", ".moov": ".mov", ".f4v": ".mp4",
		".tif": ".tiff", ".heic": ".heif",
	}
}
