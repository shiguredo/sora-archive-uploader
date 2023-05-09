package archive

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	zlog "github.com/rs/zerolog/log"
	"github.com/shiguredo/sora-archive-uploader/s3"
)

func uploadJSONFile(
	ctx context.Context,
	osConfig *s3.S3CompatibleObjectStorage,
	dst, filePath string,
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

	s3Client, err := s3.NewClient(osConfig.Endpoint, creds)
	if err != nil {
		return "", err
	}

	n, err := s3Client.FPutObject(ctx,
		osConfig.BucketName, dst, filePath,
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

	objectURL := fmt.Sprintf("s3://%s/%s", n.Bucket, n.Key)
	return objectURL, nil
}

func uploadWebMFile(ctx context.Context, osConfig *s3.S3CompatibleObjectStorage, dst, filePath string) (string, error) {
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
	s3Client, err := s3.NewClient(osConfig.Endpoint, creds)
	if err != nil {
		return "", err
	}

	zlog.Debug().
		Str("dst", dst).
		Msg("WEB-UPLOAD-START")
	n, err := s3Client.FPutObject(ctx,
		osConfig.BucketName, dst, filePath,
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

	objectURL := fmt.Sprintf("s3://%s/%s", n.Bucket, n.Key)
	return objectURL, nil
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
