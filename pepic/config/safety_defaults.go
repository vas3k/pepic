package config

func defaultSafety() SafetyConfig {
	return SafetyConfig{
		ImageThreads:             1,
		MaxConcurrentJobs:        2,
		MaxUploadFiles:           10,
		MultipartMemoryBytes:     1048576,
		MaxImagePixels:           40000000,
		MaxVideoPixels:           8294400,
		MaxDimension:             16384,
		MinResizeLength:          200,
		MaxVideoSeconds:          600,
		MaxStreams:               8,
		MaxVideoStreams:          1,
		ProbeTimeoutSeconds:      15,
		DecodeTimeoutSeconds:     30,
		TranscodeTimeoutSeconds:  120,
		ProbeOutputBytes:         65536,
		FFmpegMaxAllocationBytes: 67108864,
		FFmpegThreads:            1,
		FFmpegProtocols:          []string{"file"},
		VipsCacheBytes:           33554432,
		VipsCacheEntries:         0,
		ReadHeaderTimeoutSeconds: 10,
		ReadTimeoutSeconds:       120,
		WriteTimeoutSeconds:      300,
		IdleTimeoutSeconds:       60,
		MaxHeaderBytes:           32768,
		CookieTTLSeconds:         2592000,
	}
}
