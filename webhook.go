package archive

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"
)

type WebhookReportUploaded struct {
	ID                string          `json:"id"`
	Type              string          `json:"type"`
	Timestamp         time.Time       `json:"timestamp"`
	RecordingID       string          `json:"recording_id"`
	ChannelID         string          `json:"channel_id"`
	Filename          string          `json:"filename"`
	FileURL           string          `json:"file_url"`
	RecordingMetadata json.RawMessage `json:"recording_metadata,omitempty"`
}

type WebhookArchiveUploaded struct {
	ID               string    `json:"id"`
	Type             string    `json:"type"`
	Timestamp        time.Time `json:"timestamp"`
	RecordingID      string    `json:"recording_id"`
	SessionID        string    `json:"session_id"`
	ClientID         string    `json:"client_id"`
	ChannelID        string    `json:"channel_id"`
	ConnectionID     string    `json:"connection_id"`
	Filename         string    `json:"filename"`
	FileURL          string    `json:"file_url"`
	MetadataFilename string    `json:"metadata_filename"`
	MetadataFileURL  string    `json:"metadata_file_url"`
}

type WebhookArchiveEndUploaded struct {
	ID           string    `json:"id"`
	Type         string    `json:"type"`
	Timestamp    time.Time `json:"timestamp"`
	RecordingID  string    `json:"recording_id"`
	SessionID    string    `json:"session_id"`
	ClientID     string    `json:"client_id"`
	ChannelID    string    `json:"channel_id"`
	ConnectionID string    `json:"connection_id"`
	Filename     string    `json:"filename"`
	FileURL      string    `json:"file_url"`
}

// mTLS を組み込んだ http.Client を構築する
func createHTTPClient(config *Config) (*http.Client, error) {
	e, err := url.Parse(config.WebhookEndpointURL)
	if err != nil {
		return nil, err
	}

	// http または VerifyCacertPath 指定していない場合はそのまま投げる
	if e.Scheme != "https" || config.WebhookTLSVerifyCacertPath == "" {
		client := &http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
			Timeout: time.Duration(config.WebhookRequestTimeoutS) * time.Second,
		}

		return client, nil
	}

	CaCert, err := os.ReadFile(config.WebhookTLSVerifyCacertPath)
	if err != nil {
		return nil, err
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(CaCert)

	var certificates []tls.Certificate
	if config.WebhookTLSFullchainPath != "" && config.WebhookTLSPrivkeyPath != "" {
		pair, err := tls.LoadX509KeyPair(config.WebhookTLSFullchainPath, config.WebhookTLSPrivkeyPath)
		if err != nil {
			return nil, err
		}
		certificates = append(certificates, pair)
	}

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse

		},
		Timeout: time.Duration(config.WebhookRequestTimeoutS) * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				// hostname はチェックする
				ServerName:   e.Hostname(),
				RootCAs:      caCertPool,
				Certificates: certificates,
			},
			// TODO: config へ
			// ForceAttemptHTTP2: true,
		},
	}

	return client, nil
}

func (u Uploader) httpClientDo(client *http.Client, webhookType string, buf []byte) error {
	req, err := http.NewRequest("POST", u.config.WebhookEndpointURL, bytes.NewBuffer(buf))
	if err != nil {
		return err
	}

	// 固有ヘッダーを追加する
	req.Header.Set("Content-Type", "application/json")
	req.Header.Add(u.config.WebhookTypeHeaderName, webhookType)

	// 設定があれば Basic 認証に対応する
	if u.config.WebhookBasicAuthUsername != "" && u.config.WebhookBasicAuthPassword != "" {
		req.SetBasicAuth(u.config.WebhookBasicAuthUsername, u.config.WebhookBasicAuthPassword)
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status_code: %d", resp.StatusCode)
	}

	return nil
}

func (u Uploader) postWebhook(webhookType string, buf []byte) error {
	client, err := createHTTPClient(u.config)
	if err != nil {
		return err
	}
	if err := u.httpClientDo(client, webhookType, buf); err != nil {
		return err
	}

	return nil
}
