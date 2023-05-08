package s3

import (
	"net/url"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// config ではなくこちらにまとめる
type S3CompatibleObjectStorage struct {
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
	BucketName      string
}

func maybeEndpointURL(endpoint string) (string, bool) {
	// もし endpoint に指定されたのが endpoint_url だった場合、
	// scheme をチェックして http ならば secure = false にする
	// さらに host だけを取り出して endpoint として扱う
	var secure = false
	u, err := url.Parse(endpoint)
	// エラーがあっても無視してそのまま文字列として扱う
	// エラーがないときだけ scheme チェックする
	if err == nil {
		switch u.Scheme {
		case "http":
			return u.Host, secure
		case "https":
			// https なので secure を true にする
			secure = true
			return u.Host, secure
		case "":
			// scheme なしの場合は secure を true にする
			secure = true
			return endpoint, secure
		default:
			// サポート外の scheme の場合はタダの文字列として扱う
		}
	}
	return endpoint, secure
}

func NewClient(endpoint string, credentials *credentials.Credentials) (*minio.Client, error) {
	newEndpoint, secure := maybeEndpointURL(endpoint)
	return minio.New(
		newEndpoint,
		&minio.Options{
			Creds:  credentials,
			Secure: secure,
		})
}
