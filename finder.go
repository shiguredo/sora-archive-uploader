package archive

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	zlog "github.com/rs/zerolog/log"
)

func runFileFinder(archiveDir string) ([]string, error) {
	var result []string
	zlog.Debug().Str("archive-dir", archiveDir).Msg("START-SCRAPE-DIRECTORY")
	files, err := os.ReadDir(archiveDir)
	if err != nil {
		zlog.Err(err).Msg("ERROR-RUN-FILE-FINDER")
		return result, err
	}
	replaceWebmPattern := regexp.MustCompile(`.json$`)
	for _, f := range files {
		if !f.IsDir() {
			continue
		}
		dirPath := filepath.Join(archiveDir, f.Name())
		archiveFiles, err := os.ReadDir(dirPath)
		if err != nil {
			zlog.Err(err).Msg("ERROR-READ-DIRECTORY")
			continue
		}
		var reportFile *string
		for _, archiveFile := range archiveFiles {
			fullpath := filepath.Join(dirPath, archiveFile.Name())
			filename := archiveFile.Name()
			if !(strings.HasSuffix(filename, ".json")) {
				zlog.Debug().
					Str("file_path", fullpath).
					Msg("IGNORE-FILE-TYPE")
				continue
			}
			// 以下の処理は .json ファイルであることが保証される
			if strings.HasPrefix(filename, "report-") {
				zlog.Debug().
					Str("file_path", fullpath).
					Msg("FOUND-AT-FINDER")
				reportFile = &fullpath
			} else if strings.HasPrefix(filename, "split-archive-end-") {
				zlog.Debug().
					Str("file_path", fullpath).
					Msg("FOUND-AT-FINDER")
				result = append(result, fullpath)
			} else if strings.HasPrefix(filename, "archive-") || strings.HasPrefix(filename, "split-archive-") {
				// webm ファイルの存在を確認し、ファイルが存在したら後続の処理にファイルパスを渡す
				// webm ファイルが存在しない場合は、次回のスクレイピングのタイミングで処理する
				webmFilename := replaceWebmPattern.ReplaceAllString(filename, ".webm")
				webmFullpath := filepath.Join(dirPath, webmFilename)
				if info, err := os.Stat(webmFullpath); err != nil || info.IsDir() {
					continue
				}
				zlog.Debug().
					Str("file_path", fullpath).
					Str("webm_file_path", webmFullpath).
					Msg("FOUND-AT-FINDER")
				result = append(result, fullpath)
			} else {
				zlog.Debug().
					Str("file_path", fullpath).
					Msg("IGNORE-FILE")
			}
		}
		// ディレクトリ内に report json ファイルが見つかった場合は、最後に流す
		if reportFile != nil {
			result = append(result, *reportFile)
		}
	}
	zlog.Debug().Str("archive-dir", archiveDir).Msg("END-SCRAPE-DIRECTORY")
	return result, nil
}
