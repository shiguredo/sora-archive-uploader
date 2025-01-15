package archive

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shiguredo/sora-archive-uploader/s3"
	base32 "github.com/shogo82148/go-clockwork-base32"

	zlog "github.com/rs/zerolog/log"
)

type RecordingReport struct {
	RecordingID       string          `json:"recording_id"`
	ChannelID         string          `json:"channel_id"`
	SessionID         string          `json:"session_id"`
	FilePath          string          `json:"file_path"`
	Filename          string          `json:"filename"`
	Metadata          json.RawMessage `json:"metadata"`
	RecordingMetadata json.RawMessage `json:"recording_metadata"`
}

type UploaderManager struct {
	ArchiveStream    chan UploaderResult
	ArchiveEndStream chan UploaderResult
	ReportStream     chan UploaderResult
	uploaders        []Uploader
}

type ArchiveMetadata struct {
	RecordingID      string `json:"recording_id"`
	ChannelID        string `json:"channel_id"`
	SessionID        string `json:"session_id"`
	ClientID         string `json:"client_id"`
	ConnectionID     string `json:"connection_id"`
	FilePath         string `json:"file_path"`
	Filename         string `json:"filename"`
	MetadataFilePath string `json:"metadata_file_path"`
	MetadataFilename string `json:"metadata_filename"`
}

type ArchiveEndMetadata struct {
	RecordingID  string `json:"recording_id"`
	ChannelID    string `json:"channel_id"`
	SessionID    string `json:"session_id"`
	ClientID     string `json:"client_id"`
	ConnectionID string `json:"connection_id"`
	FilePath     string `json:"file_path"`
	Filename     string `json:"filename"`
}

func newUploaderManager() *UploaderManager {
	var uploaders []Uploader
	archiveStream := make(chan UploaderResult)
	archiveEndStream := make(chan UploaderResult)
	reportStream := make(chan UploaderResult)
	return &UploaderManager{
		ArchiveStream:    archiveStream,
		ArchiveEndStream: archiveEndStream,
		ReportStream:     reportStream,
		uploaders:        uploaders,
	}
}

func (um *UploaderManager) run(ctx context.Context, config *Config, fileStream <-chan string) (*UploaderManager, error) {
	for i := 0; i < config.UploadWorkers; i++ {
		uploader, err := newUploader(i+1, config)
		if err != nil {
			return nil, err
		}
		uploader.run(fileStream, um.ArchiveStream, um.ArchiveEndStream, um.ReportStream)
		um.uploaders = append(um.uploaders, *uploader)
	}
	go func() {
		defer func() {
			close(um.ArchiveStream)
			close(um.ArchiveEndStream)
			close(um.ReportStream)
		}()

		<-ctx.Done()
		zlog.Debug().Msg("STOP-UPLOADER-MANAGER")
		for _, u := range um.uploaders {
			u.Stop()
		}
		zlog.Debug().Msg("STOPPED-UPLOADER-MANAGER")
	}()
	return um, nil
}

type UploaderResult struct {
	Success  bool
	Filepath string
}

type Uploader struct {
	id            int
	config        *Config
	ctx           context.Context
	cancel        context.CancelFunc
	base32Encoder *base32.Encoding
}

func newUploader(id int, config *Config) (*Uploader, error) {
	u := &Uploader{
		id:            id,
		config:        config,
		base32Encoder: base32.NewEncoding(),
	}
	u.ctx, u.cancel = context.WithCancel(context.Background())
	return u, nil
}

func (u Uploader) run(
	fileStream <-chan string,
	outArchive chan UploaderResult,
	outArchiveEnd chan UploaderResult,
	outReport chan UploaderResult,
) {
	go func() {
		for {
			select {
			case <-u.ctx.Done():
				zlog.Debug().
					Int("uploader_id", u.id).
					Msg("STOPPED-UPLOADER")
				return
			case inputFilepath, ok := <-fileStream:
				if !ok {
					continue
				}
				filename := filepath.Base(inputFilepath)
				if strings.HasPrefix(filename, "report-") {
					zlog.Debug().
						Int("uploader_id", u.id).
						Str("json_file_path", inputFilepath).
						Msg("FOUND-AT-STARTUP")
					ok := u.handleReport(inputFilepath)
					select {
					case <-u.ctx.Done():
						return
					case outReport <- UploaderResult{
						Success:  ok,
						Filepath: inputFilepath,
					}:
					}
				} else if strings.HasPrefix(filename, "split-archive-end-") {
					zlog.Debug().
						Int("uploader_id", u.id).
						Str("json_file_path", inputFilepath).
						Msg("FOUND-AT-STARTUP")
					ok := u.handleArchiveEnd(inputFilepath)
					select {
					case <-u.ctx.Done():
						return
					case outArchiveEnd <- UploaderResult{
						Success:  ok,
						Filepath: inputFilepath,
					}:
					}
				} else if strings.HasPrefix(filename, "archive-") {
					zlog.Debug().
						Int("uploader_id", u.id).
						Str("file_path", inputFilepath).
						Msg("FOUND-AT-STARTUP")
					ok := u.handleArchive(inputFilepath, false)
					select {
					case <-u.ctx.Done():
						return
					case outArchive <- UploaderResult{
						Success:  ok,
						Filepath: inputFilepath,
					}:
					}
				} else if strings.HasPrefix(filename, "split-archive-") {
					zlog.Debug().
						Int("uploader_id", u.id).
						Str("file_path", inputFilepath).
						Msg("FOUND-AT-STARTUP")
					ok := u.handleArchive(inputFilepath, true)
					select {
					case <-u.ctx.Done():
						return
					case outArchive <- UploaderResult{
						Success:  ok,
						Filepath: inputFilepath,
					}:
					}
				}
			}
		}
	}()
}

func (u Uploader) Stop() {
	u.cancel()
}

func (u Uploader) handleArchive(archiveJSONFilePath string, split bool) bool {
	fileInfo, err := os.Stat(archiveJSONFilePath)
	if err != nil {
		zlog.Error().
			Err(err).
			Msg("JSON-NOT-ACCESSIBLE")
		return false
	}

	// json をパースする
	raw, err := os.ReadFile(archiveJSONFilePath)
	if err != nil {
		zlog.Error().
			Err(err).
			Int("uploader_id", u.id).
			Str("path", archiveJSONFilePath).
			Msg("ARCHIVE-JSON-FILE-READ-ERROR")
		return false
	}

	var am ArchiveMetadata
	if err := json.Unmarshal(raw, &am); err != nil {
		zlog.Error().
			Err(err).
			Int("uploader_id", u.id).
			Str("path", archiveJSONFilePath).
			Msg("ARCHIVE-JSON-PARSE-ERROR")
		return false
	}

	// ここで s3 ファイルをアップロード
	// json がくればファイルパスもわかる
	zlog.Debug().
		Int("uploader_id", u.id).
		Str("recording_id", am.RecordingID).
		Str("channel_id", am.ChannelID).
		Str("connection_id", am.ConnectionID).
		Msg("ARCHIVE-METADATA-INFO")

	// webm ファイルのパスを作っておく
	webmFilename := filepath.Base(am.Filename)
	webmFilepath := filepath.Join(filepath.Dir(archiveJSONFilePath), webmFilename)

	// metadata ファイル (json) をアップロード
	metadataFilename := fileInfo.Name()
	metadataObjectKey := fmt.Sprintf("%s/%s", am.RecordingID, metadataFilename)
	osConfig := &s3.S3CompatibleObjectStorage{
		Endpoint:        u.config.ObjectStorageEndpoint,
		BucketName:      u.config.ObjectStorageBucketName,
		AccessKeyID:     u.config.ObjectStorageAccessKeyID,
		SecretAccessKey: u.config.ObjectStorageSecretAccessKey,
	}

	// webm ファイルを開いておく
	f, err := os.Open(webmFilepath)
	if err != nil {
		zlog.Error().
			Err(err).
			Int("uploader_id", u.id).
			Str("path", archiveJSONFilePath).
			Str("webm_filename", am.FilePath).
			Msg("WEBM-FILE-OPEN-ERROR")
		return false
	}
	defer f.Close()

	zlog.Info().
		Str("path", webmFilepath).
		Msg("WEBM-FILE-PATH")

	metadataFileURL, err := uploadJSONFile(
		u.ctx,
		osConfig,
		metadataObjectKey,
		archiveJSONFilePath,
	)
	if err != nil {
		zlog.Error().
			Err(err).
			Int("uploader_id", u.id).
			Str("path", archiveJSONFilePath).
			Str("metadata_filename", metadataFilename).
			Str("metadata_object_key", metadataObjectKey).
			Msg("METADATA-FILE-UPLOAD-ERROR")
		if !isFileContinuous(err) {
			// リトライしないエラーの場合は、ファイルを削除
			u.removeArchiveJSONFile(archiveJSONFilePath, webmFilepath)
			u.removeArchiveWEBMFile(archiveJSONFilePath, webmFilepath)
		}
		return false
	}
	zlog.Debug().
		Int("uploader_id", u.id).
		Str("uploaded_matadata", am.MetadataFilename).
		Msg("UPLOAD-METADATA-FILE-SUCCESSFULLY")

	webmObjectKey := fmt.Sprintf("%s/%s", am.RecordingID, webmFilename)

	var fileURL string
	if u.config.UploadFileRateLimitMbps == 0 {
		fileURL, err = uploadWebMFile(u.ctx, osConfig, webmObjectKey, webmFilepath)
	} else {
		fileURL, err = uploadWebMFileWithRateLimit(u.ctx, osConfig, webmObjectKey, webmFilepath, u.config.UploadFileRateLimitMbps)
	}

	if err != nil {
		zlog.Error().
			Err(err).
			Int("uploader_id", u.id).
			Str("path", archiveJSONFilePath).
			Str("webm_filename", webmFilename).
			Str("webm_object_key", webmObjectKey).
			Msg("WEBM-FILE-UPLOAD-ERROR")
		if !isFileContinuous(err) {
			// リトライしないエラーの場合は、ファイルを削除
			u.removeArchiveJSONFile(archiveJSONFilePath, webmFilepath)
			u.removeArchiveWEBMFile(archiveJSONFilePath, webmFilepath)
		}
		return false
	}
	zlog.Debug().
		Int("uploader_id", u.id).
		Str("uploaded_webm", am.Filename).
		Msg("UPLOAD-WEBM-FILE-SUCCESSFULLY")

	if u.config.WebhookEndpointURL != "" {
		var archiveUploadedType string
		if split {
			archiveUploadedType = u.config.WebhookTypeSplitArchiveUploaded
		} else {
			archiveUploadedType = u.config.WebhookTypeArchiveUploaded
		}
		webhookID, err := u.generateWebhookID()
		if err != nil {
			zlog.Error().
				Err(err).
				Int("uploader_id", u.id).
				Str("uploaded_webm", am.Filename).
				Msg("WEBHOOK-ID-GENERATE-ERROR")
			return false
		}
		var w = WebhookArchiveUploaded{
			ID:               webhookID,
			Type:             archiveUploadedType,
			Timestamp:        time.Now().UTC(),
			SessionID:        am.SessionID,
			ClientID:         am.ClientID,
			RecordingID:      am.RecordingID,
			ChannelID:        am.ChannelID,
			ConnectionID:     am.ConnectionID,
			Filename:         webmFilename,
			FileURL:          fileURL,
			MetadataFilename: metadataFilename,
			MetadataFileURL:  metadataFileURL,
		}
		buf, err := json.Marshal(w)
		if err != nil {
			zlog.Error().
				Int("uploader_id", u.id).
				Err(err).
				Msg("ARCHIVE-UPLOADED-WEBHOOK-MARSHAL-ERROR")
			return false
		}
		if err := u.postWebhook(
			archiveUploadedType,
			buf,
		); err != nil {
			zlog.Error().
				Err(err).
				Int("uploader_id", u.id).
				Str("recording_id", w.RecordingID).
				Str("channel_id", w.ChannelID).
				Str("filename", w.Filename).
				Str("metadata_filename", w.MetadataFilename).
				Msg("ARCHIVE-UPLOADED-WEBHOOK-SEND-ERROR")
			return false
		}
	}

	// 処理し終わったファイルを削除
	jsonError := u.removeArchiveJSONFile(archiveJSONFilePath, webmFilepath)
	webmError := u.removeArchiveWEBMFile(archiveJSONFilePath, webmFilepath)
	return jsonError == nil && webmError == nil
}

func (u Uploader) handleReport(reportJSONFilePath string) bool {
	fileInfo, err := os.Stat(reportJSONFilePath)
	if err != nil {
		zlog.Error().
			Err(err).
			Msg("JSON-NOT-ACCESSABLE")
		return false
	}

	// report- ファイルのアップロード
	// json をパースする
	raw, err := os.ReadFile(reportJSONFilePath)
	if err != nil {
		zlog.Error().
			Err(err).
			Int("uploader_id", u.id).
			Str("path", reportJSONFilePath).
			Msg("REPORT-JSON-FILE-READ-ERROR")
		return false
	}
	var rr RecordingReport
	if err := json.Unmarshal(raw, &rr); err != nil {
		zlog.Error().
			Err(err).
			Int("uploader_id", u.id).
			Str("path", reportJSONFilePath).
			Msg("REPORT-JSON-FILE-UNMARSHAL-ERROR")
		return false
	}

	// report ファイル (json) をアップロード
	filename := fileInfo.Name()
	reportObjectKey := fmt.Sprintf("%s/%s", rr.RecordingID, filename)
	osConfig := &s3.S3CompatibleObjectStorage{
		Endpoint:        u.config.ObjectStorageEndpoint,
		BucketName:      u.config.ObjectStorageBucketName,
		AccessKeyID:     u.config.ObjectStorageAccessKeyID,
		SecretAccessKey: u.config.ObjectStorageSecretAccessKey,
	}

	fileURL, err := uploadJSONFile(
		u.ctx,
		osConfig,
		reportObjectKey,
		reportJSONFilePath,
	)
	if err != nil {
		zlog.Error().
			Err(err).
			Int("uploader_id", u.id).
			Str("filename", filename).
			Str("report_object_key", reportObjectKey).
			Msg("REPORT-FILE-UPLOAD-ERROR")
		if !isFileContinuous(err) {
			// リトライしないエラーの場合は、ファイルを削除
			u.removeReportFile(reportJSONFilePath)
		}
		return false
	}
	zlog.Debug().
		Int("uploader_id", u.id).
		Str("uplaoded_report", filename).
		Msg("UPLOAD-REPORT-JSON-SUCCESSFULLY")

	if u.config.WebhookEndpointURL != "" {
		webhookID, err := u.generateWebhookID()
		if err != nil {
			zlog.Error().
				Err(err).
				Int("uploader_id", u.id).
				Str("uploaded_report", filename).
				Msg("WEBHOOK-ID-GENERATE-ERROR")
			return false
		}
		var w = WebhookReportUploaded{
			ID:          webhookID,
			Type:        u.config.WebhookTypeReportUploaded,
			Timestamp:   time.Now().UTC(),
			RecordingID: rr.RecordingID,
			ChannelID:   rr.ChannelID,
			Filename:    filename,
			FileURL:     fileURL,
		}

		// recording_metadata の除外設定が *無効* の時は recording_metadata をウェブフックに含める
		// 関数は !config.ExcludeWebhookRecordingMetadata の値を返しています
		if u.config.IncludeWebhookRecordingMetadata() {
			// セッション録画とレガシー録画では、録画の metadata のキーが異なるための分岐
			// SessionID が空でなければセッション録画とみなす
			if rr.SessionID != "" {
				w.RecordingMetadata = rr.RecordingMetadata
			} else {
				w.RecordingMetadata = rr.Metadata
			}
		}

		buf, err := json.Marshal(w)
		if err != nil {
			zlog.Error().
				Err(err).
				Int("uploader_id", u.id).
				Str("recording_id", w.RecordingID).
				Str("channel_id", w.ChannelID).
				Str("filename", w.Filename).
				Msg("REPORT-UPLOAD-WEBHOOK-MARSHAL-ERROR")
			return false
		}
		if err := u.postWebhook(
			u.config.WebhookTypeReportUploaded,
			buf,
		); err != nil {
			zlog.Error().
				Err(err).
				Int("uploader_id", u.id).
				Str("recording_id", w.RecordingID).
				Str("channel_id", w.ChannelID).
				Str("filename", w.Filename).
				Msg("REPORT-UPLOADED-WEBHOOK-SEND-ERROR")
			return false
		}
		zlog.Debug().
			Int("uploader_id", u.id).
			Str("recording_id", w.RecordingID).
			Str("channel_id", w.ChannelID).
			Str("filename", w.Filename).
			Msg("REPORT-UPLOADED-WEBHOOK-SEND-SUCCESSFULLY")
	}

	// 処理し終わったファイルを削除
	if err = u.removeReportFile(reportJSONFilePath); err != nil {
		return false
	}
	return true
}

func (u Uploader) handleArchiveEnd(archiveEndJSONFilePath string) bool {
	fileInfo, err := os.Stat(archiveEndJSONFilePath)
	if err != nil {
		zlog.Error().
			Err(err).
			Msg("JSON-NOT-ACCESSIBLE")
		return false
	}

	// json をパースする
	raw, err := os.ReadFile(archiveEndJSONFilePath)
	if err != nil {
		zlog.Error().
			Err(err).
			Int("uploader_id", u.id).
			Str("path", archiveEndJSONFilePath).
			Msg("archive json file read error")
		return false
	}

	var aem ArchiveEndMetadata
	if err := json.Unmarshal(raw, &aem); err != nil {
		zlog.Error().
			Err(err).
			Int("uploader_id", u.id).
			Str("path", archiveEndJSONFilePath).
			Msg("ARCHIVE-END-JSON-FILE-PARSE-ERROR")
		return false
	}

	zlog.Debug().
		Int("uploader_id", u.id).
		Str("recording_id", aem.RecordingID).
		Str("channel_id", aem.ChannelID).
		Str("connection_id", aem.ConnectionID).
		Msg("ARCHIVE-END-METADATA-INFO")

	// metadata ファイル (json) をアップロード
	filename := fileInfo.Name()
	objectKey := fmt.Sprintf("%s/%s", aem.RecordingID, filename)
	osConfig := &s3.S3CompatibleObjectStorage{
		Endpoint:        u.config.ObjectStorageEndpoint,
		BucketName:      u.config.ObjectStorageBucketName,
		AccessKeyID:     u.config.ObjectStorageAccessKeyID,
		SecretAccessKey: u.config.ObjectStorageSecretAccessKey,
	}

	archiveEndURL, err := uploadJSONFile(
		u.ctx,
		osConfig,
		objectKey,
		archiveEndJSONFilePath,
	)
	if err != nil {
		zlog.Error().
			Err(err).
			Int("uploader_id", u.id).
			Str("path", archiveEndJSONFilePath).
			Str("filename", filename).
			Str("object_key", objectKey).
			Msg("METADATA-FILE-UPLOAD-ERROR")
		if !isFileContinuous(err) {
			u.removeArchiveEndFile(archiveEndJSONFilePath)
		}
		return false
	}
	zlog.Debug().
		Int("uploader_id", u.id).
		Str("uploaded_archive_end", aem.Filename).
		Str("archive_end_presigned_url", archiveEndURL).
		Msg("UPLOAD-ARCHIVE-END-FILE-SUCCESSFULLY")

	if u.config.WebhookEndpointURL != "" {
		webhookID, err := u.generateWebhookID()
		if err != nil {
			zlog.Error().
				Err(err).
				Int("uploader_id", u.id).
				Str("uploaded_archive_end", aem.Filename).
				Str("archive_end_presigned_url", archiveEndURL).
				Msg("WEBHOOK-ID-GENERATE-ERROR")
			return false
		}
		var w = WebhookArchiveEndUploaded{
			ID:           webhookID,
			Type:         u.config.WebhookTypeSplitArchiveEndUploaded,
			Timestamp:    time.Now().UTC(),
			RecordingID:  aem.RecordingID,
			SessionID:    aem.SessionID,
			ClientID:     aem.ClientID,
			ChannelID:    aem.ChannelID,
			ConnectionID: aem.ConnectionID,
			Filename:     filename,
			FileURL:      archiveEndURL,
		}
		buf, err := json.Marshal(w)
		if err != nil {
			zlog.Error().
				Int("uploader_id", u.id).
				Err(err).
				Msg("ARCHIVE-UPLOADED-WEBHOOK-MARSHAL-ERROR")
			return false
		}
		if err := u.postWebhook(
			u.config.WebhookTypeSplitArchiveEndUploaded,
			buf,
		); err != nil {
			zlog.Error().
				Err(err).
				Int("uploader_id", u.id).
				Str("recording_id", w.RecordingID).
				Str("channel_id", w.ChannelID).
				Str("filename", w.Filename).
				Msg("ARCHIVE-END-UPLOADED-WEBHOOK-SEND-ERROR")
			return false
		}
	}

	if err = u.removeArchiveEndFile(archiveEndJSONFilePath); err != nil {
		return false
	}
	return true
}

func (u Uploader) removeArchiveJSONFile(metadataFilePath, webmFilepath string) error {
	err := os.Remove(metadataFilePath)
	if err != nil {
		zlog.Error().
			Err(err).
			Int("uploader_id", u.id).
			Str("metadata_filepath", metadataFilePath).
			Str("archive_filepath", webmFilepath).
			Msg("FAILED-REMOVE-METADATA-JSON-FILE")
	} else {
		zlog.Debug().
			Int("uploader_id", u.id).
			Str("metadata_filepath", metadataFilePath).
			Str("archive_filepath", webmFilepath).
			Msg("REMOVED-METADATA-JSON-FILE")
	}
	return err
}

func (u Uploader) removeArchiveWEBMFile(metadataFilePath, webmFilepath string) error {
	err := os.Remove(webmFilepath)
	if err != nil {
		zlog.Error().
			Err(err).
			Int("uploader_id", u.id).
			Str("archive_filepath", webmFilepath).
			Msg("FAILED-REMOVE-ARCHIVE-WEBM-FILE")
	} else {
		zlog.Debug().
			Int("uploader_id", u.id).
			Str("archive_filepath", webmFilepath).
			Msg("remove archive webm file successfully.")
	}
	return err
}

func (u Uploader) removeReportFile(reportJSONFilePath string) error {
	err := os.Remove(reportJSONFilePath)
	if err != nil {
		zlog.Error().
			Err(err).
			Int("uploader_id", u.id).
			Str("filepath", reportJSONFilePath).
			Msg("FAILED-REMOVE-REPORT-JSON-FILE")
	} else {
		zlog.Debug().
			Int("uploader_id", u.id).
			Str("filepath", reportJSONFilePath).
			Msg("REMOVED-REPORT-JSON-FILE")

	}
	return err
}

func (u Uploader) removeArchiveEndFile(archiveEndJSONFilePath string) error {
	err := os.Remove(archiveEndJSONFilePath)
	if err != nil {
		zlog.Error().
			Err(err).
			Int("uploader_id", u.id).
			Str("filepath", archiveEndJSONFilePath).
			Msg("FAILED-REMOVE-ARCHIVE-END-FILE")
	} else {
		zlog.Debug().
			Int("uploader_id", u.id).
			Str("filepath", archiveEndJSONFilePath).
			Msg("REMOVED-ARCHIVE-END-FILE")
	}
	return err
}

func (u Uploader) generateWebhookID() (string, error) {
	id := uuid.New()
	binaryUUID, err := id.MarshalBinary()
	if err != nil {
		return "", err
	}
	return u.base32Encoder.EncodeToString(binaryUUID), nil
}
