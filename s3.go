package archive

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	zlog "github.com/rs/zerolog/log"
	"github.com/shiguredo/sora-archive-uploader/s3"

	"github.com/conduitio/bwlimit"
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

	zlog.Info().
		Str("dst", dst).
		Msg("WEB-UPLOAD-START")
	n, err := s3Client.FPutObject(ctx,
		osConfig.BucketName, dst, filePath,
		minio.PutObjectOptions{ContentType: "application/octet-stream"},
	)
	if err != nil {
		return "", err
	}
	zlog.Info().
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

func uploadWebMFileWithRateLimit(ctx context.Context, osConfig *s3.S3CompatibleObjectStorage, dst, filePath string,
	rateLimitMpbs int) (string, error) {
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

	// bit を byte にする
	rateLimitMByteps := (bwlimit.Byte(rateLimitMpbs) * bwlimit.MiB) / 8

	// 受信には制限をかけない
	dialer := bwlimit.NewDialer(&net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}, rateLimitMByteps, 0)

	transport, err := s3.DefaultTransport(osConfig.Endpoint)
	if err != nil {
		return "", err
	}
	transport.DialContext = dialer.DialContext

	s3Client, err := s3.NewClientWithTransport(osConfig.Endpoint, creds, transport)
	if err != nil {
		return "", err
	}

	fileReader, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer fileReader.Close()

	// Save the file stat.
	fileStat, err := fileReader.Stat()
	if err != nil {
		return "", err
	}

	// Save the file size.
	fileSize := fileStat.Size()

	zlog.Info().
		Str("dst", dst).
		Msg("WEB-UPLOAD-START")

	// 使用帯域の制限時は、巨大なサイズのファイルのアップロードする時に使用される multipart アップロードで
	// 並列アップロードは行わずに 1 thread で処理されるようにオプションを設定する
	n, err := s3Client.PutObject(ctx, osConfig.BucketName, dst, fileReader, fileSize,
		minio.PutObjectOptions{ContentType: "application/octet-stream", NumThreads: 1})
	if err != nil {
		return "", err
	}

	zlog.Info().
		Str("dst", dst).
		Int64("size", n.Size).
		Msg("UPLOAD-WEBM-SUCCESSFULLY")

	objectURL := fmt.Sprintf("s3://%s/%s", n.Bucket, n.Key)
	return objectURL, nil
}
