package archive

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	zlog "github.com/rs/zerolog/log"
)

// config ではなくこちらにまとめる
type S3CompatibleObjectStorage struct {
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
	BucketName      string
}

func uploadJSONFile(
	ctx context.Context,
	osConfig *S3CompatibleObjectStorage,
	reader io.Reader, size int64, dst string,
) (string, error) {
	var creds *credentials.Credentials
	if (osConfig.AccessKeyID != "") || (osConfig.SecretAccessKey != "") {
		creds = credentials.NewStaticV4(
			osConfig.AccessKeyID,
			osConfig.SecretAccessKey,
			"",
		)
	} else if (len(os.Getenv("AWS_ACCESS_KEY_ID")) > 0) && (len(os.Getenv("AWS_SECRET_ACCESS_KEY")) > 0) {
		creds = credentials.NewEnvAWS()
	} else {
		creds = credentials.NewIAM("")
	}

	s3Client, err := newS3Client(osConfig.Endpoint, creds)
	if err != nil {
		return "", err
	}

	n, err := s3Client.PutObject(ctx,
		osConfig.BucketName, dst,
		reader, size,
		minio.PutObjectOptions{ContentType: "application/octet-stream"},
	)
	if err != nil {
		return "", err
	}
	zlog.Debug().
		Str("dst", dst).
		Int64("size", n.Size).
		Msg("UPLAOD-SUCCESSFULLY")

	reqParams := make(url.Values)
	filename := filepath.Base(dst)
	zlog.Debug().
		Str("filename", filename).
		Msg("CREATE-CONTENT-DISPOSITION-FILENAME")
	reqParams.Set(
		"response-content-disposition",
		fmt.Sprintf("attachment; filename=\"%s\"", filename),
	)

	objectUrl := fmt.Sprintf("s3://%s/%s", n.Bucket, n.Key)
	return objectUrl, nil
}

func uploadWebMFile(ctx context.Context, osConfig *S3CompatibleObjectStorage, file *os.File, dst string) (string, error) {
	var creds *credentials.Credentials
	if (osConfig.AccessKeyID != "") || (osConfig.SecretAccessKey != "") {
		creds = credentials.NewStaticV4(
			osConfig.AccessKeyID,
			osConfig.SecretAccessKey,
			"",
		)
	} else if (len(os.Getenv("AWS_ACCESS_KEY_ID")) > 0) && (len(os.Getenv("AWS_SECRET_ACCESS_KEY")) > 0) {
		creds = credentials.NewEnvAWS()
	} else {
		creds = credentials.NewIAM("")
	}
	s3Client, err := newS3Client(osConfig.Endpoint, creds)
	if err != nil {
		return "", err
	}

	fileStat, err := file.Stat()
	if err != nil {
		return "", err
	}

	zlog.Debug().
		Str("dst", dst).
		Msg("WEB-UPLOAD-START")
	n, err := s3Client.PutObject(ctx,
		osConfig.BucketName, dst,
		file, fileStat.Size(),
		minio.PutObjectOptions{ContentType: "application/octet-stream"},
	)
	if err != nil {
		return "", err
	}
	zlog.Debug().
		Str("dst", dst).
		Int64("size", n.Size).
		Msg("UPLOAD-WEBM-SUCCESSFULLY")

	reqParams := make(url.Values)
	filename := filepath.Base(dst)
	zlog.Debug().
		Str("filename", filename).
		Msg("create content-disposition filename")
	reqParams.Set(
		"response-content-disposition",
		fmt.Sprintf("attachment; filename=\"%s\"", filename),
	)

	objectUrl := fmt.Sprintf("s3://%s/%s", n.Bucket, n.Key)
	return objectUrl, nil
}

// minio のエラーをレスポンスに復元して、リトライするためファイルを残すか対象のファイルを削除するか判断する
func isFileContinuous(err error) bool {
	errResp := minio.ToErrorResponse(err)
	switch errResp.Code {
	case "NoSuchBucket":
		return false
	case "AccessDenied":
		return false
	case "InvalidRegion":
		return false
	}
	return true
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

func newS3Client(endpoint string, credentials *credentials.Credentials) (*minio.Client, error) {
	newEndpoint, secure := maybeEndpointURL(endpoint)
	s3Client, err := minio.New(
		newEndpoint,
		&minio.Options{
			Creds:  credentials,
			Secure: secure,
		})
	if err != nil {
		return nil, err
	}

	return s3Client, nil
}
