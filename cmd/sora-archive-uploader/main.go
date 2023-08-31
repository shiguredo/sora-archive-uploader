package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	archive "github.com/shiguredo/sora-archive-uploader"
)

var (
	version   string
	revision  string
	buildDate string

	versionText = `sora-archive-uploader build info.
version: %s
revision: %s
build date: %s
`
)

func main() {
	configFilePath := flag.String("C", "config.ini", "Config file path")
	var v bool
	flag.BoolVar(&v, "version", false, "Show version")
	flag.Parse()

	if v {
		fmt.Printf(versionText, version, revision, buildDate)
		os.Exit(0)
	}

	log.Printf("sora-archive-uploader version:%s revision:%s build_date:%s", version, revision, buildDate)
	log.Printf("config file path: %s", *configFilePath)
	archive.Run(configFilePath)
}
