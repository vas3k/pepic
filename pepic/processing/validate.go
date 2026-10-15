package processing

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/davidbyttow/govips/v2/vips"
	"github.com/vas3k/pepic/pepic/config"
)

var ErrInvalidMedia = errors.New("unsupported or invalid media")

func ValidateMedia(data []byte) (string, error) {
	return ValidateMediaContext(context.Background(), data)
}

func ValidateMediaContext(ctx context.Context, data []byte) (string, error) {
	if len(data) == 0 {
		return "", ErrInvalidMedia
	}
	mimeType, format, ok := detectMedia(data)
	if !ok {
		return "", ErrInvalidMedia
	}
	switch format.Processor {
	case "vips":
		image, err := loadImage(data)
		if err != nil {
			return "", ErrInvalidMedia
		}
		defer image.Close()
		if err := image.Resize(1/float64(max(image.Width(), image.Height())), vips.KernelLanczos3); err != nil {
			return "", ErrInvalidMedia
		}
		if _, err := image.ToBytes(); err != nil {
			return "", ErrInvalidMedia
		}

	case "ffmpeg":
		if format.GoDecode {
			cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
			if err != nil || !validDimensions(cfg.Width, cfg.Height, config.App.Safety.MaxVideoPixels) {
				return "", ErrInvalidMedia
			}
		}
		if err := validateVideo(ctx, data, format); err != nil {
			return "", err
		}
	default:
		return "", ErrInvalidMedia
	}
	return mimeType, nil
}

func detectMedia(data []byte) (string, config.MediaFormat, bool) {
	imageType := vips.DetermineImageType(data)
	for mimeType, format := range config.App.Media.Formats {
		if format.ImageType != vips.ImageTypeUnknown && format.ImageType == imageType {
			return mimeType, format, true
		}
		if len(format.Signatures) == 0 {
			continue
		}
		matches := true
		for _, signature := range format.Signatures {
			magic, err := hex.DecodeString(signature.Hex)
			offset := signature.Offset
			if err != nil || len(magic) == 0 || offset < 0 || offset > len(data) || len(magic) > len(data)-offset || !bytes.Equal(data[offset:offset+len(magic)], magic) {
				matches = false
				break
			}
		}
		if matches {
			return mimeType, format, true
		}
	}
	return config.App.Media.Format(http.DetectContentType(data))
}

func validDimensions(w, h, pixels int) bool {
	maxDimension := config.App.Safety.MaxDimension
	return w > 0 && h > 0 && w <= maxDimension && h <= maxDimension && int64(w)*int64(h) <= int64(pixels)
}

func mediaInputArgs(format config.MediaFormat) []string {
	safety := config.App.Safety
	args := []string{
		"-v", "error",
		"-max_alloc", strconv.Itoa(safety.FFmpegMaxAllocationBytes),
		"-protocol_whitelist", strings.Join(safety.FFmpegProtocols, ","),
		"-threads", strconv.Itoa(safety.FFmpegThreads),
		"-f", format.Demuxer,
	}
	return append(args, format.InputArgs...)
}

type cappedOutput struct{ bytes.Buffer }

func (b *cappedOutput) Write(p []byte) (int, error) {
	if b.Len()+len(p) > config.App.Safety.ProbeOutputBytes {
		return 0, errors.New("media probe output too large")
	}
	return b.Buffer.Write(p)
}

func validateVideo(parent context.Context, data []byte, format config.MediaFormat) error {
	dir, err := os.MkdirTemp(config.App.Videos.FFmpeg.TempDir, "pepic-probe-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	input := dir + "/input"
	if err = os.WriteFile(input, data, 0600); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(parent, time.Duration(config.App.Safety.ProbeTimeoutSeconds)*time.Second)
	defer cancel()
	args := mediaInputArgs(format)
	args = append(args,
		"-show_entries", "stream=codec_type,codec_name,width,height,nb_frames:stream_disposition=attached_pic:format=duration",
		"-of", "json",
		input,
	)
	cmd := exec.CommandContext(ctx, "ffprobe", args...)
	var output cappedOutput
	cmd.Stdout = &output
	if err = cmd.Run(); err != nil {
		return fmt.Errorf("%w: cannot probe media", ErrInvalidMedia)
	}
	var info struct {
		Streams []struct {
			CodecType   string `json:"codec_type"`
			CodecName   string `json:"codec_name"`
			Frames      string `json:"nb_frames"`
			Width       int    `json:"width"`
			Height      int    `json:"height"`
			Disposition struct {
				AttachedPic int `json:"attached_pic"`
			} `json:"disposition"`
		} `json:"streams"`
		Format struct {
			Duration string `json:"duration"`
		} `json:"format"`
	}
	if json.Unmarshal(output.Bytes(), &info) != nil || len(info.Streams) > config.App.Safety.MaxStreams {
		return ErrInvalidMedia
	}
	videos := 0
	for _, s := range info.Streams {
		if s.CodecType == "video" {
			if s.Disposition.AttachedPic != 0 || s.CodecName == "" || !validDimensions(s.Width, s.Height, config.App.Safety.MaxVideoPixels) {
				return ErrInvalidMedia
			}
			videos++
		} else if s.CodecType != "audio" {
			return ErrInvalidMedia
		}
	}
	duration, err := strconv.ParseFloat(info.Format.Duration, 64)
	validStreams := videos >= 1 && videos <= config.App.Safety.MaxVideoStreams
	validDuration := err == nil && duration > 0 && duration <= float64(config.App.Safety.MaxVideoSeconds)
	if !validDuration && format.AllowStillImage && len(info.Streams) == 1 && info.Streams[0].Frames == "1" && (err != nil || duration == 0) {
		validDuration = true
	}
	if !validStreams || !validDuration {
		return ErrInvalidMedia
	}
	ctxDecode, cancelDecode := operationTimeout(parent, config.App.Safety.DecodeTimeoutSeconds)
	defer cancelDecode()
	decodeArgs := append(mediaInputArgs(format), "-nostdin", "-xerror")
	decodeArgs = append(decodeArgs,
		"-i", input,
		"-map", "0:v",
		"-an",
		"-threads", strconv.Itoa(config.App.Safety.FFmpegThreads),
		"-f", "null", "-",
	)
	if err := exec.CommandContext(ctxDecode, "ffmpeg", decodeArgs...).Run(); err != nil {
		return ErrInvalidMedia
	}
	return nil
}
