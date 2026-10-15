package config

import (
	"fmt"
	"math"

	"github.com/dustin/go-humanize"
)

type SafetyConfig struct {
	ImageThreads             int      `yaml:"image_threads"`
	MaxConcurrentJobs        int      `yaml:"max_concurrent_jobs"`
	MaxUploadFiles           int      `yaml:"max_upload_files"`
	MultipartMemoryBytes     int64    `yaml:"multipart_memory_bytes"`
	MaxImagePixels           int      `yaml:"max_image_pixels"`
	MaxVideoPixels           int      `yaml:"max_video_pixels"`
	MaxDimension             int      `yaml:"max_dimension"`
	MinResizeLength          int      `yaml:"min_resize_length"`
	MaxVideoSeconds          int      `yaml:"max_video_seconds"`
	MaxStreams               int      `yaml:"max_streams"`
	MaxVideoStreams          int      `yaml:"max_video_streams"`
	ProbeTimeoutSeconds      int      `yaml:"probe_timeout_seconds"`
	DecodeTimeoutSeconds     int      `yaml:"decode_timeout_seconds"`
	TranscodeTimeoutSeconds  int      `yaml:"transcode_timeout_seconds"`
	ProbeOutputBytes         int      `yaml:"probe_output_bytes"`
	FFmpegMaxAllocationBytes int      `yaml:"ffmpeg_max_allocation_bytes"`
	FFmpegThreads            int      `yaml:"ffmpeg_threads"`
	FFmpegProtocols          []string `yaml:"ffmpeg_protocols"`
	VipsCacheBytes           int      `yaml:"vips_cache_bytes"`
	VipsCacheEntries         int      `yaml:"vips_cache_entries"`
	ReadHeaderTimeoutSeconds int      `yaml:"read_header_timeout_seconds"`
	ReadTimeoutSeconds       int      `yaml:"read_timeout_seconds"`
	WriteTimeoutSeconds      int      `yaml:"write_timeout_seconds"`
	IdleTimeoutSeconds       int      `yaml:"idle_timeout_seconds"`
	MaxHeaderBytes           int      `yaml:"max_header_bytes"`
	CookieTTLSeconds         int      `yaml:"cookie_ttl_seconds"`
}

func UploadLimit() (int64, error) {
	n, err := humanize.ParseBytes(App.Global.MaxUploadSize)
	if err != nil || n <= 0 || n >= math.MaxInt64 {
		return 0, fmt.Errorf("global.max_upload_size must be a positive byte size")
	}
	return int64(n), nil
}

func Validate() error {
	if App.Images.PNGCompression < -3 || App.Images.PNGCompression > 9 {
		return fmt.Errorf("invalid PNG compression")
	}
	if App.Images.JPEGQuality < 1 || App.Images.JPEGQuality > 100 {
		return fmt.Errorf("invalid image quality")
	}
	if _, err := UploadLimit(); err != nil {
		return err
	}
	if err := App.Media.Validate(); err != nil {
		return err
	}
	s := App.Safety
	positive := map[string]int{
		"image_threads":               s.ImageThreads,
		"max_concurrent_jobs":         s.MaxConcurrentJobs,
		"max_upload_files":            s.MaxUploadFiles,
		"max_image_pixels":            s.MaxImagePixels,
		"max_video_pixels":            s.MaxVideoPixels,
		"max_dimension":               s.MaxDimension,
		"min_resize_length":           s.MinResizeLength,
		"max_video_seconds":           s.MaxVideoSeconds,
		"max_streams":                 s.MaxStreams,
		"max_video_streams":           s.MaxVideoStreams,
		"probe_timeout_seconds":       s.ProbeTimeoutSeconds,
		"decode_timeout_seconds":      s.DecodeTimeoutSeconds,
		"transcode_timeout_seconds":   s.TranscodeTimeoutSeconds,
		"probe_output_bytes":          s.ProbeOutputBytes,
		"ffmpeg_max_allocation_bytes": s.FFmpegMaxAllocationBytes,
		"ffmpeg_threads":              s.FFmpegThreads,
		"read_header_timeout_seconds": s.ReadHeaderTimeoutSeconds,
		"read_timeout_seconds":        s.ReadTimeoutSeconds,
		"write_timeout_seconds":       s.WriteTimeoutSeconds,
		"idle_timeout_seconds":        s.IdleTimeoutSeconds,
		"max_header_bytes":            s.MaxHeaderBytes,
		"cookie_ttl_seconds":          s.CookieTTLSeconds,
	}
	for key, value := range positive {
		if value <= 0 {
			return fmt.Errorf("safety.%s must be positive", key)
		}
	}
	if s.MultipartMemoryBytes <= 0 || s.VipsCacheBytes < 0 || s.VipsCacheEntries < 0 || len(s.FFmpegProtocols) == 0 {
		return fmt.Errorf("invalid safety memory/cache/protocol settings")
	}
	if s.MaxVideoStreams > s.MaxStreams || s.MinResizeLength > s.MaxDimension {
		return fmt.Errorf("inconsistent safety limits")
	}
	if App.Global.FileTreeSplitChars <= 0 {
		return fmt.Errorf("file_tree_split_chars must be positive")
	}
	for _, target := range []struct{ mime, processor string }{
		{App.Images.AutoConvert, "vips"},
		{App.Images.GIFConvert, "ffmpeg"},
		{App.Videos.AutoConvert, "ffmpeg"},
	} {
		if target.mime == "" || target.mime == "false" {
			continue
		}
		_, f, ok := App.Media.Format(target.mime)
		if !ok || f.Processor != target.processor {
			return fmt.Errorf("conversion target %q is not an allowed %s format", target.mime, target.processor)
		}
	}
	return nil
}
