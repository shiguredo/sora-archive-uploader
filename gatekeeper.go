package archive

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"

	zlog "github.com/rs/zerolog/log"
)

type RecordingUnit struct {
	mutex       sync.RWMutex
	recordingID string
	counter     int32
	reportFile  string
}

func newRecordingUnit(recordingID string) *RecordingUnit {
	return &RecordingUnit{
		recordingID: recordingID,
		counter:     0,
	}
}

func (ru *RecordingUnit) run() {
	ru.mutex.Lock()
	defer ru.mutex.Unlock()
	atomic.AddInt32(&ru.counter, 1)
	zlog.Debug().
		Str("recording_id", ru.recordingID).
		Int32("counter", ru.counter).
		Msgf("RUN-RECORDING-UNIT")
}

func (ru *RecordingUnit) done() {
	atomic.AddInt32(&ru.counter, -1)
	zlog.Debug().
		Str("recording_id", ru.recordingID).
		Int32("counter", ru.counter).
		Msgf("DONE-RECORDING-UNIT")
}

func (ru *RecordingUnit) canProcessAndSetReportFile(reportFile string) bool {
	ru.mutex.Lock()
	defer ru.mutex.Unlock()
	if atomic.LoadInt32(&ru.counter) == 0 {
		return true
	}
	ru.reportFile = reportFile
	return false
}

func (ru *RecordingUnit) canProcessAndGetReportFile() (*string, bool) {
	ru.mutex.Lock()
	defer ru.mutex.Unlock()
	if atomic.LoadInt32(&ru.counter) == 0 && ru.reportFile != "" {
		return &ru.reportFile, true
	}
	return nil, false
}

type GateKeeper struct {
	mutex             sync.RWMutex
	config            *Config
	ctx               context.Context
	processingList    sync.Map
	processingCounter int64
	out               chan string
}

func newGateKeeper(config *Config) *GateKeeper {
	g := &GateKeeper{
		config:         config,
		processingList: sync.Map{},
		out:            make(chan string, 50),
	}
	return g
}

func (g *GateKeeper) stop() {
	close(g.out)
	zlog.Debug().Msg("STOPPED-GATE-KEEPER")
}

func (g *GateKeeper) run(ctx context.Context, infiles []string) <-chan string {
	g.ctx = ctx
	go func() {
		defer g.stop()

		processArchiveFile := func(infile string) {
			filename := filepath.Base(infile)
			recordingID := filepath.Base(filepath.Dir(infile))
			atomic.AddInt64(&g.processingCounter, 1)
			if strings.HasPrefix(filename, "report-") {
				g.mutex.Lock()
				ru, ok := g.getRecordingUnit(recordingID)
				if !ok {
					// report-* の前に他のファイルが処理されてない
					g.mutex.Unlock()
					go func() {
						select {
						case <-g.ctx.Done():
						case g.out <- infile:
						}
					}()
					return
				}
				g.mutex.Unlock()

				if ru.canProcessAndSetReportFile(infile) {
					go func() {
						select {
						case <-g.ctx.Done():
						case g.out <- infile:
						}
					}()
					return
				}
				return
			}
			if strings.HasPrefix(filename, "split-archive-end-") {
				g.processRun(recordingID)
				select {
				case <-g.ctx.Done():
				case g.out <- infile:
				}
				return
			}
			if strings.HasPrefix(filename, "archive-") || strings.HasPrefix(filename, "split-archive-") {
				archiveID := strings.Split(filename, ".")[0]
				g.processRun(recordingID)
				select {
				case <-g.ctx.Done():
					return
				case g.out <- infile:
					zlog.Debug().Str("archive_id", archiveID).Msg("RUN-ARCHIVE-FILE-PROCESS")
				}
				return
			}
		}

		for _, infile := range infiles {
			processArchiveFile(infile)
		}
		<-g.ctx.Done()
	}()
	return g.out
}

func (g *GateKeeper) getRecordingUnit(recordingID string) (*RecordingUnit, bool) {
	ru, ok := g.processingList.Load(recordingID)
	if ok {
		return ru.(*RecordingUnit), ok
	} else {
		return nil, ok
	}
}

func (g *GateKeeper) processRun(recordingID string) {
	g.mutex.Lock()
	defer g.mutex.Unlock()
	ru, ok := g.getRecordingUnit(recordingID)
	if !ok {
		ru = newRecordingUnit(recordingID)
		g.processingList.Store(recordingID, ru)
	}
	ru.run()
}

func (g *GateKeeper) processDone(infile string) {
	g.mutex.Lock()
	defer g.mutex.Unlock()
	zlog.Debug().Str("infile", infile).Msg("PROCESS-DONE")

	recordingID := filepath.Base(filepath.Dir(infile))
	ru, ok := g.getRecordingUnit(recordingID)
	if !ok {
		atomic.AddInt64(&g.processingCounter, -1)
		zlog.Error().Str("infile", infile).Msg("WAIT-GROUP-NOT-FOUND")
		return
	}
	ru.done()
	if reportFile, ok := ru.canProcessAndGetReportFile(); ok {
		go func() {
			select {
			case <-g.ctx.Done():
			case g.out <- *reportFile:
			}
		}()
	}
	atomic.AddInt64(&g.processingCounter, -1)
}

func (g *GateKeeper) recordingDone(infile string) {
	zlog.Debug().Str("infile", infile).Msg("RECORDING-DONE")

	// Recording ID のディレクトリを削除するべきだが、いきなり削除せず、mv で監視対象外のパスに移動しておく
	dirname := filepath.Dir(infile)
	newDirPath := filepath.Join(g.config.SoraEvacuateDirFullPath, filepath.Base(dirname))
	var evacuatePath = g.config.SoraEvacuateDirFullPath
	_, err := os.Stat(g.config.SoraEvacuateDirFullPath)
	if err != nil {
		err = os.Mkdir(evacuatePath, 0755)
		if err != nil {
			zlog.Error().
				Str("evacuate_dir_path", evacuatePath).
				Str("old_path", dirname).
				Str("new_path", newDirPath).
				Msg("EVACUATE-DIRECTORY-CREATE-ERROR")
		}
	}
	err = os.Rename(dirname, newDirPath)
	if err != nil {
		zlog.Error().
			Err(err).
			Str("old_path", dirname).
			Str("new_path", newDirPath).
			Msg("RECORDING-DIRECTORY-MOVE-ERROR")
	} else {
		zlog.Debug().
			Str("old_path", dirname).
			Str("new_path", newDirPath).
			Msg("RECORDING-DIRECTORY-MOVE-SUCCESSFULLY")
	}
	atomic.AddInt64(&g.processingCounter, -1)
}

func (g *GateKeeper) isFileUploadFinished() bool {
	return atomic.LoadInt64(&g.processingCounter) == 0
}
