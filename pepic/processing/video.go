package processing

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"

	"github.com/vas3k/pepic/pepic/config"
	"github.com/vas3k/pepic/pepic/entity"
	"github.com/vas3k/pepic/pepic/utils"
)

type VideoBackend interface {
	Transcode(file *entity.ProcessingFile, maxLength int) error
	Convert(file *entity.ProcessingFile, newMimeType string) error
}

type videoBackend struct{}

func NewVideoBackend() VideoBackend { return &videoBackend{} }

func (v *videoBackend) Transcode(file *entity.ProcessingFile, maxLength int) error {
	return v.process(file, file.Mime, maxLength)
}

func (v *videoBackend) Convert(file *entity.ProcessingFile, mime string) error {
	return v.process(file, mime, 0)
}

func (v *videoBackend) process(file *entity.ProcessingFile, mime string, length int) error {
	_, format, ok := config.App.Media.Format(file.Mime)
	if !ok || format.Processor != "ffmpeg" {
		return ErrInvalidMedia
	}
	canonicalMIME, outputFormat, ok := config.App.Media.Format(mime)
	if !ok || outputFormat.Processor != "ffmpeg" {
		return ErrInvalidMedia
	}
	ext, err := utils.ExtensionByMimeType(canonicalMIME)
	if err != nil {
		return err
	}
	dir, err := os.MkdirTemp(config.App.Videos.FFmpeg.TempDir, "pepic-transcode-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	input, output := filepath.Join(dir, "input"), filepath.Join(dir, "output"+ext)
	if err := os.WriteFile(input, file.Bytes, 0600); err != nil {
		return err
	}
	limit, err := config.UploadLimit()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(config.App.Safety.TranscodeTimeoutSeconds)*time.Second)
	defer cancel()
	args := append(mediaInputArgs(format), "-nostdin", "-xerror", "-filter_threads", strconv.Itoa(config.App.Safety.FFmpegThreads))
	args = append(args,
		"-i", input,
		"-map", "0:v:0",
		"-map", "0:a:0?",
		"-map_metadata", "-1",
		"-map_chapters", "-1",
		"-threads", strconv.Itoa(config.App.Safety.FFmpegThreads),
		"-t", strconv.Itoa(config.App.Safety.MaxVideoSeconds),
		"-fs", strconv.FormatInt(limit, 10),
	)
	cfg := config.App.Videos.FFmpeg
	args = append(args,
		"-c:v", cfg.VideoCodec,
		"-preset", cfg.Preset,
		"-crf", strconv.Itoa(cfg.CRF),
		"-b:v", cfg.VideoBitrate,
		"-profile:v", cfg.VideoProfile,
		"-c:a", cfg.AudioCodec,
		"-b:a", cfg.AudioBitrate,
		"-bufsize", strconv.Itoa(cfg.BufferSize),
		"-pix_fmt", cfg.PixFmt,
	)
	if outputFormat.UseMovFlags {
		args = append(args, "-movflags", cfg.MovFlags)
	}
	if length > 0 {
		args = append(args, "-vf", fmt.Sprintf("scale=w='min(%d,iw)':h='min(%d,ih)':force_original_aspect_ratio=decrease:force_divisible_by=2", length, length))
	}
	args = append(args, output)
	if err := exec.CommandContext(ctx, "ffmpeg", args...).Run(); err != nil {
		return fmt.Errorf("video processing failed: %w", err)
	}
	f, err := os.Open(output)
	if err != nil {
		return err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return err
	}
	if info.Size() <= 0 || info.Size() >= limit {
		return fmt.Errorf("processed video exceeds size limit")
	}
	data, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil {
		return err
	}
	if int64(len(data)) > limit {
		return fmt.Errorf("processed video exceeds size limit")
	}
	file.Bytes, file.Mime = data, canonicalMIME
	file.Filename = utils.ReplaceExt(file.Filename, ext)
	if file.Path != "" {
		file.Path = utils.ReplaceExt(file.Path, ext)
	}
	return nil
}
