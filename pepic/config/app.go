package config

type Config struct {
	Global struct {
		Host               string `yaml:"host" env:"HOST" env-default:"0.0.0.0"`
		Port               int    `yaml:"port" env:"PORT" env-default:"8118"`
		BaseUrl            string `yaml:"base_url" env:"BASE_URL" env-default:"http://0.0.0.0:8118/"`
		SecretCode         string `yaml:"secret_code" env:"SECRET_CODE" env-default:""`
		MaxUploadSize      string `yaml:"max_upload_size" env:"MAX_UPLOAD_SIZE" env-default:""`
		FileTreeSplitChars int    `yaml:"file_tree_split_chars" env:"FILE_TREE_SPLIT_CHARS" env-default:"3"`
	} `yaml:"global"`

	Storage struct {
		Type string `yaml:"type" env:"STORAGE_TYPE" env-default:"provider"`
		Dir  string `yaml:"dir" env:"STORAGE_DIR" env-default:"uploads/"`
	} `yaml:"storage"`

	Images struct {
		StoreOriginals bool   `yaml:"store_originals" env:"IMAGE_STORE_ORIGINALS" env-default:"false"`
		OriginalLength int    `yaml:"original_length" env:"IMAGE_ORIGINAL_LENGTH" env-default:"1800"`
		LiveResize     bool   `yaml:"live_resize" env:"IMAGE_LIVE_RESIZE" env-default:"true"`
		AutoConvert    string `yaml:"auto_convert" env:"IMAGE_AUTO_CONVERT" env-default:"false"`
		JPEGQuality    int    `yaml:"jpeg_quality" env:"IMAGE_JPEG_QUALITY" env-default:"95"`
		PNGCompression int    `yaml:"png_compression" env:"IMAGE_PNG_COMPRESSION" env-default:"0"`
	} `yaml:"images"`

	Videos struct {
		StoreOriginals bool   `yaml:"store_originals" env:"VIDEO_STORE_ORIGINALS" env-default:"false"`
		OriginalLength int    `yaml:"original_length" env:"VIDEO_ORIGINAL_LENGTH" env-default:"480"`
		LiveResize     bool   `yaml:"live_resize" env:"VIDEO_LIVE_RESIZE" env-default:"false"`
		AutoConvert    string `yaml:"auto_convert" env:"VIDEO_AUTO_CONVERT" env-default:"video/mp4"`
		FFmpeg         struct {
			TempDir      string `yaml:"temp_dir" env:"VIDEO_FFMPEG_TEMP_DIR" env-default:"/tmp"`
			Preset       string `yaml:"preset" env:"VIDEO_FFMPEG_PRESET" env-default:"slow"`
			CRF          int    `yaml:"crf" env:"VIDEO_FFMPEG_CRF" env-default:"24"`
			BufferSize   int    `yaml:"buffer_size" env:"VIDEO_FFMPEG_BUFFER_SIZE" env-default:"1024000"`
			VideoCodec   string `yaml:"video_codec" env:"VIDEO_FFMPEG_VIDEO_CODEC" env-default:"libx264"`
			VideoBitrate string `yaml:"video_bitrate" env:"VIDEO_FFMPEG_VIDEO_BITRATE" env-default:"1024k"`
			VideoProfile string `yaml:"video_profile" env:"VIDEO_FFMPEG_VIDEO_PROFILE" env-default:"main"`
			AudioCodec   string `yaml:"audio_codec" env:"VIDEO_FFMPEG_AUDIO_CODEC" env-default:"aac"`
			AudioBitrate string `yaml:"audio_bitrate" env:"VIDEO_FFMPEG_AUDIO_BITRATE" env-default:"128k"`
			MovFlags     string `yaml:"mov_flags" env:"VIDEO_FFMPEG_MOV_FLAGS" env-default:"+faststart"`
			PixFmt       string `yaml:"pix_fmt" env:"VIDEO_FFMPEG_PIX_FMT" env-default:"yuv420p"`
		} `yaml:"ffmpeg"`
	} `yaml:"videos"`

	Meta struct {
		Blocks []struct {
			Title    string `yaml:"title"`
			Template string `yaml:"template"`
		} `yaml:"blocks" env:"META_BLOCKS"`
	} `yaml:"meta"`
}

var App Config
