package config

type Config struct {
	Jobs                JobsConfig        `yaml:"jobs"`
	CanonicalExtensions map[string]string `yaml:"canonical_extensions"`
	Media               MediaConfig       `yaml:"media"`
	Safety              SafetyConfig      `yaml:"safety"`
	Global              struct {
		Host               string `yaml:"host" env:"HOST" envDefault:"0.0.0.0"`
		Port               int    `yaml:"port" env:"PORT" envDefault:"8118"`
		BaseUrl            string `yaml:"base_url" env:"BASE_URL" envDefault:"http://0.0.0.0:8118/"`
		SecretCode         string `yaml:"secret_code" env:"SECRET_CODE" envDefault:""`
		MaxUploadSize      string `yaml:"max_upload_size" env:"MAX_UPLOAD_SIZE" envDefault:"50M"`
		FileTreeSplitChars int    `yaml:"file_tree_split_chars" env:"FILE_TREE_SPLIT_CHARS" envDefault:"3"`
	} `yaml:"global"`

	Storage struct {
		Type string `yaml:"type" env:"STORAGE_TYPE" envDefault:"fs"`
		Dir  string `yaml:"dir" env:"STORAGE_DIR" envDefault:"uploads/"`
	} `yaml:"storage"`

	Images struct {
		StoreOriginals bool   `yaml:"store_originals" env:"IMAGE_STORE_ORIGINALS" envDefault:"false"`
		OriginalLength int    `yaml:"original_length" env:"IMAGE_ORIGINAL_LENGTH" envDefault:"1800"`
		LiveResize     bool   `yaml:"live_resize" env:"IMAGE_LIVE_RESIZE" envDefault:"true"`
		AutoConvert    string `yaml:"auto_convert" env:"IMAGE_AUTO_CONVERT" envDefault:"false"`
		JPEGQuality    int    `yaml:"jpeg_quality" env:"IMAGE_JPEG_QUALITY" envDefault:"95"`
		PNGCompression int    `yaml:"png_compression" env:"IMAGE_PNG_COMPRESSION" envDefault:"6"`
		GIFConvert     string `yaml:"gif_convert" env:"IMAGE_GIF_CONVERT"`
	} `yaml:"images"`

	Videos struct {
		StoreOriginals bool   `yaml:"store_originals" env:"VIDEO_STORE_ORIGINALS" envDefault:"false"`
		OriginalLength int    `yaml:"original_length" env:"VIDEO_ORIGINAL_LENGTH" envDefault:"480"`
		LiveResize     bool   `yaml:"live_resize" env:"VIDEO_LIVE_RESIZE" envDefault:"false"`
		AutoConvert    string `yaml:"auto_convert" env:"VIDEO_AUTO_CONVERT"`
		FFmpeg         struct {
			TempDir      string `yaml:"temp_dir" env:"VIDEO_FFMPEG_TEMP_DIR" envDefault:"/tmp"`
			Preset       string `yaml:"preset" env:"VIDEO_FFMPEG_PRESET" envDefault:"slow"`
			CRF          int    `yaml:"crf" env:"VIDEO_FFMPEG_CRF" envDefault:"24"`
			BufferSize   int    `yaml:"buffer_size" env:"VIDEO_FFMPEG_BUFFER_SIZE" envDefault:"1024000"`
			VideoCodec   string `yaml:"video_codec" env:"VIDEO_FFMPEG_VIDEO_CODEC" envDefault:"libx264"`
			VideoBitrate string `yaml:"video_bitrate" env:"VIDEO_FFMPEG_VIDEO_BITRATE" envDefault:"1024k"`
			VideoProfile string `yaml:"video_profile" env:"VIDEO_FFMPEG_VIDEO_PROFILE" envDefault:"main"`
			AudioCodec   string `yaml:"audio_codec" env:"VIDEO_FFMPEG_AUDIO_CODEC" envDefault:"aac"`
			AudioBitrate string `yaml:"audio_bitrate" env:"VIDEO_FFMPEG_AUDIO_BITRATE" envDefault:"128k"`
			MovFlags     string `yaml:"mov_flags" env:"VIDEO_FFMPEG_MOV_FLAGS" envDefault:"+faststart"`
			PixFmt       string `yaml:"pix_fmt" env:"VIDEO_FFMPEG_PIX_FMT" envDefault:"yuv420p"`
		} `yaml:"ffmpeg"`
	} `yaml:"videos"`

	Meta struct {
		ImageTemplates []struct {
			Title    string `yaml:"title"`
			Template string `yaml:"template"`
		} `yaml:"image_templates" env:"META_IMAGE_TEMPLATES"`

		VideoTemplates []struct {
			Title    string `yaml:"title"`
			Template string `yaml:"template"`
		} `yaml:"video_templates" env:"META_VIDEO_TEMPLATES"`

		MultiTemplates []struct {
			Title    string `yaml:"title"`
			Template string `yaml:"template"`
		} `yaml:"multi_templates" env:"META_MULTI_TEMPLATES"`
	} `yaml:"meta"`
}

var App = Defaults()
