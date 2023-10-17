package archive

import (
	_ "embed"

	"gopkg.in/ini.v1"
)

//go:embed VERSION
var Version string

const (
	DefaultLogDir  = "."
	DefaultLogName = "sora-archive-uploader.jsonl"

	// megabytes
	DefaultLogRotateMaxSize    = 200
	DefaultLogRotateMaxBackups = 7
	// days
	DefaultLogRotateMaxAge = 30
)

type Config struct {
	Debug bool `ini:"debug"`

	LogDir    string `ini:"log_dir"`
	LogName   string `ini:"log_name"`
	LogStdout bool   `ini:"log_stdout"`

	LogRotateMaxSize    int  `ini:"log_rotate_max_size"`
	LogRotateMaxBackups int  `ini:"log_rotate_max_backups"`
	LogRotateMaxAge     int  `ini:"log_rotate_max_age"`
	LogRotateCompress   bool `ini:"log_rotate_compress"`

	ObjectStorageEndpoint        string `ini:"object_storage_endpoint"`
	ObjectStorageBucketName      string `ini:"object_storage_bucket_name"`
	ObjectStorageAccessKeyID     string `ini:"object_storage_access_key_id"`
	ObjectStorageSecretAccessKey string `ini:"object_storage_secret_access_key"`

	SoraArchiveDirFullPath  string `ini:"archive_dir_full_path"`
	SoraEvacuateDirFullPath string `ini:"evacuate_dir_full_path"`

	UploadWorkers int `ini:"upload_workers"`

	// 1 ファイルあたりのアップロードレート制限
	UploadFileRateLimitMbps int `ini:"upload_file_rate_limit_mbps"`

	UploadedFileCacheSize int `ini:"uploaded_file_cache_size"`

	WebhookEndpointURL            string `ini:"webhook_endpoint_url"`
	WebhookEndpointHealthCheckURL string `ini:"webhook_endpoint_health_check_url"`

	WebhookTypeHeaderName              string `ini:"webhook_type_header_name"`
	WebhookTypeArchiveUploaded         string `ini:"webhook_type_archive_uploaded"`
	WebhookTypeSplitArchiveUploaded    string `ini:"webhook_type_split_archive_uploaded"`
	WebhookTypeSplitArchiveEndUploaded string `ini:"webhook_type_split_archive_end_uploaded"`
	WebhookTypeReportUploaded          string `ini:"webhook_type_report_uploaded"`

	WebhookBasicAuthUsername string `ini:"webhook_basic_auth_username"`
	WebhookBasicAuthPassword string `ini:"webhook_basic_auth_password"`

	WebhookRequestTimeoutS int32 `ini:"webhook_request_timeout_s"`

	WebhookTLSVerifyCacertPath string `ini:"webhook_tls_verify_cacert_path"`
	WebhookTLSFullchainPath    string `ini:"webhook_tls_fullchain_path"`
	WebhookTLSPrivkeyPath      string `ini:"webhook_tls_privkey_path"`
}

func newConfig(configFilePath string) (*Config, error) {
	config := new(Config)
	iniConfig, err := ini.InsensitiveLoad(configFilePath)
	if err != nil {
		return nil, err
	}
	if err := iniConfig.StrictMapTo(config); err != nil {
		return nil, err
	}
	return config, nil
}
