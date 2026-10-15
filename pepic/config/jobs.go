package config

import "fmt"

type JobsConfig struct {
	MaxRecordBytes           int64  `yaml:"max_record_bytes" env:"JOBS_MAX_RECORD_BYTES"`
	Dir                      string `yaml:"dir" env:"JOBS_DIR"`
	Workers                  int    `yaml:"workers" env:"JOBS_WORKERS"`
	MaxPending               int    `yaml:"max_pending" env:"JOBS_MAX_PENDING"`
	MaxRecords               int    `yaml:"max_records" env:"JOBS_MAX_RECORDS"`
	MaxBytes                 int64  `yaml:"max_bytes" env:"JOBS_MAX_BYTES"`
	RetentionSeconds         int    `yaml:"retention_seconds" env:"JOBS_RETENTION_SECONDS"`
	PollSeconds              int    `yaml:"poll_seconds" env:"JOBS_POLL_SECONDS"`
	ProcessingTimeoutSeconds int    `yaml:"processing_timeout_seconds" env:"JOBS_PROCESSING_TIMEOUT_SECONDS"`
	ShutdownTimeoutSeconds   int    `yaml:"shutdown_timeout_seconds" env:"JOBS_SHUTDOWN_TIMEOUT_SECONDS"`
	MaxAttempts              int    `yaml:"max_attempts" env:"JOBS_MAX_ATTEMPTS"`
}

func defaultJobs() JobsConfig {
	return JobsConfig{MaxRecordBytes: 65536, Workers: 1, MaxPending: 20, MaxRecords: 1000, MaxBytes: 536870912, RetentionSeconds: 86400, PollSeconds: 2, ProcessingTimeoutSeconds: 1800, ShutdownTimeoutSeconds: 30, MaxAttempts: 3}
}

func (j JobsConfig) Validate() error {
	if j.MaxRecordBytes <= 0 || j.Workers <= 0 || j.Workers > App.Safety.MaxConcurrentJobs || j.MaxPending < j.Workers || j.MaxRecords < j.MaxPending || j.MaxBytes <= 0 || j.RetentionSeconds <= 0 || j.PollSeconds <= 0 || j.ProcessingTimeoutSeconds <= 0 || j.ShutdownTimeoutSeconds <= 0 || j.MaxAttempts <= 0 {
		return fmt.Errorf("invalid jobs limits")
	}
	return nil
}
