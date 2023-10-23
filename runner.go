package archive

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	zlog "github.com/rs/zerolog/log"
)

type Main struct {
	config *Config
}

func newMain(config *Config) *Main {
	return &Main{
		config: config,
	}
}

func (m *Main) run(ctx context.Context, cancel context.CancelFunc) error {
	var archiveDir = m.config.SoraArchiveDirFullPath
	zlog.Debug().Str("path", archiveDir).Msg("WATCHING-ROOT-DIR")
	fileInfo, err := os.Stat(archiveDir)
	if err != nil {
		// 対象のディレクトリが存在しなければ終わる
		zlog.Fatal().Err(err).Str("path", archiveDir).Msg("NOT-FOUND-TARGET-PATH")
	}
	if !fileInfo.IsDir() {
		// 対象のパスが Directory でなければ終わる
		zlog.Fatal().Str("path", archiveDir).Msg("TARGET-PATH-DOES-NOT-DIRECTORY")
	}
	// TODO: パーミッションチェック

	// ディレクトリ退避先を作成する
	var evacuatePath = m.config.SoraEvacuateDirFullPath
	_, err = os.Stat(m.config.SoraEvacuateDirFullPath)
	if err != nil {
		err = os.Mkdir(evacuatePath, 0755)
		if err != nil {
			zlog.Fatal().
				Str("evacuate_dir_path", evacuatePath).
				Msg("CANT-CREATE-DIRECTORY")
		}
	}

	foundFiles, err := runFileFinder(archiveDir)
	if err != nil {
		return err
	}
	if len(foundFiles) == 0 {
		// 処理対象のファイルが見つからなかったので終わる
		cancel()
		zlog.Debug().Msg("ARCHIVE-FILE-NOT-FOUND")
		return nil
	}

	processContext, processContextCancel := context.WithCancel(context.Background())
	gateKeeper := newGateKeeper(m.config)
	recordingFileStream := gateKeeper.run(processContext, foundFiles)

	uploaderManager := newUploaderManager()
	_, err = uploaderManager.run(processContext, m.config, recordingFileStream)
	if err != nil {
		processContextCancel()
		return err
	}

	for {
		select {
		case <-ctx.Done():
			processContextCancel()
			// 停止ログ出力待ちのため、500ms 待ってから停止している
			<-time.After(500 * time.Millisecond)
			return nil
		case archiveFileResult := <-uploaderManager.ArchiveStream:
			if !archiveFileResult.Success {
				zlog.Warn().
					Str("archive_file", archiveFileResult.Filepath).
					Msg("FAILED-UPLOAD-ARCHIVE-END")
			}
			// zlog.Info().
			// 	Str("archive_file", archiveFileResult.Filepath).
			// 	Msg("UPLOADED-ARCHIVE-FILE")
			gateKeeper.processDone(archiveFileResult.Filepath)
			if gateKeeper.isFileUploadFinished() {
				cancel()
			}
		case archiveEndFileResult := <-uploaderManager.ArchiveEndStream:
			if !archiveEndFileResult.Success {
				zlog.Warn().
					Str("archive_end_file", archiveEndFileResult.Filepath).
					Msg("FAILED-UPLOAD-ARCHIVE-END")
			}
			// zlog.Info().
			// 	Str("archive_end_file", archiveEndFileResult.Filepath).
			// 	Msg("UPLOADED-ARCHIVE-END-FILE")
			gateKeeper.processDone(archiveEndFileResult.Filepath)
			if gateKeeper.isFileUploadFinished() {
				cancel()
			}
		case reportFileResult := <-uploaderManager.ReportStream:
			if !reportFileResult.Success {
				zlog.Warn().
					Str("report_file", reportFileResult.Filepath).
					Msg("FAILED-UPLOAD-ARCHIVE-END")
			}
			// zlog.Info().
			// 	Str("report_file", reportFileResult.Filepath).
			// 	Msg("UPLOADED-REPORT-FILE")
			gateKeeper.recordingDone(reportFileResult.Filepath)
			if gateKeeper.isFileUploadFinished() {
				cancel()
			}
		}
	}
}

func Run(configFilePath *string) {
	// INI をパース
	config, err := newConfig(*configFilePath)
	if err != nil {
		// パースに失敗した場合 Fatal で終了
		log.Fatal("cannot parse config file, err=", err)
	}

	// ロガー初期化
	err = initLogger(config)
	if err != nil {
		// ロガー初期化に失敗したら Fatal で終了
		log.Fatal("cannot parse config file, err=", err)
	}

	// もしあれば mTLS の設定確認と Webhook のヘルスチェック
	if config.WebhookEndpointHealthCheckURL != "" {
		client, err := createHTTPClient(config)
		if err != nil {
			zlog.Fatal().Err(err).Msg("FAILED-CREATE-RPC-CLIENT")
		}
		// ヘルスチェック URL で起動確認する
		resp, err := client.Get(config.WebhookEndpointHealthCheckURL)
		if err != nil {
			zlog.Fatal().Err(err).Msg("WEBHOOK-SERVER-CONNECT-ERROR")
		}
		if resp.StatusCode != 200 {
			zlog.Fatal().Err(err).Msg("WEBHOOK-SERVER-UNHEALTHY")
		}
		resp.Body.Close()
	}

	zlog.Debug().Msg("STARTED-SORA-ARCHIVE-UPLOADER")

	// シグナルをキャッチして停止処理
	trapSignals := []os.Signal{
		syscall.SIGINT,
		syscall.SIGTERM,
	}
	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, trapSignals...)
	ctx, cancel := context.WithCancel(context.Background())
	doneShutdown := make(chan interface{})
	defer close(doneShutdown)
	go func() {
		sig := <-signalChannel
		zlog.Debug().Str("signal", sig.String()).Msg("RECEIVED-SIGNAL")

		cancel()
		doneShutdown <- struct{}{}
	}()

	// ディレクトリ監視とアップロード処理
	m := newMain(config)
	if err := m.run(ctx, cancel); err != nil {
		zlog.Error().Err(err).Msg("FAILED-RUN")
		os.Exit(1)
	} else {
		go func() {
			doneShutdown <- struct{}{}
		}()
	}
	<-doneShutdown
	zlog.Debug().Msg("STOPPED-SORA-ARCHIVE-UPLOADER")
}
