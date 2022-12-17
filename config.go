package archive

import (
	"github.com/BurntSushi/toml"
)

type Config struct {
	Debug bool `toml:"debug"`

	LogDir              string `toml:"log_dir"`
	LogName             string `toml:"log_name"`
	LogStdOut           bool   `toml:"log_std_out"`
	LogRotateMaxSize    int    `toml:"log_rotate_max_size"`
	LogRotateMaxBackups int    `toml:"log_rotate_max_backups"`
	LogRotateMaxAge     int    `toml:"log_rotate_max_age"`
	LogRotateCompress   bool   `toml:"log_rotate_compress"`

	ObjectStorageEndpoint        string `toml:"object_storage_endpoint"`
	ObjectStorageBucketName      string `toml:"object_storage_bucket_name"`
	ObjectStorageAccessKeyID     string `toml:"object_storage_access_key_id"`
	ObjectStorageSecretAccessKey string `toml:"object_storage_secret_access_key"`

	SoraArchiveDirFullPath  string `toml:"archive_dir_full_path"`
	SoraEvacuateDirFullPath string `toml:"evacuate_dir_full_path"`

	UploadWorkers int `toml:"upload_workers"`

	UploadedFileCacheSize int `toml:"uploaded_file_cache_size"`

	WebhookEndpointURL            string `toml:"webhook_endpoint_url"`
	WebhookEndpointHealthCheckURL string `toml:"webhook_endpoint_health_check_url"`

	WebhookTypeHeaderName              string `toml:"webhook_type_header_name"`
	WebhookTypeArchiveUploaded         string `toml:"webhook_type_archive_uploaded"`
	WebhookTypeSplitArchiveUploaded    string `toml:"webhook_type_split_archive_uploaded"`
	WebhookTypeSplitArchiveEndUploaded string `toml:"webhook_type_split_archive_end_uploaded"`
	WebhookTypeReportUploaded          string `toml:"webhook_type_report_uploaded"`

	WebhookBasicAuthUsername string `toml:"webhook_basic_auth_username"`
	WebhookBasicAuthPassword string `toml:"webhook_basic_auth_password"`

	WebhookRequestTimeoutS int32 `toml:"webhook_request_timeout_s"`

	WebhookTlsVerifyCacertPath string `toml:"webhook_tls_verify_cacert_path"`
	WebhookTlsFullchainPath    string `toml:"webhook_tls_fullchain_path"`
	WebhookTlsPrivkeyPath      string `toml:"webhook_tls_privkey_path"`
}

func initConfig(data []byte, config interface{}) error {
	if err := toml.Unmarshal(data, config); err != nil {
		return err
	}

	// TODO: 初期値
	return nil
}
