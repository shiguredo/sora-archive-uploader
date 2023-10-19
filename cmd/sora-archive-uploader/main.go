package main

import (
	"flag"
	"fmt"
	"log"

	archive "github.com/shiguredo/sora-archive-uploader"
)

func main() {
	// /bin/sora-archive-uploader -V
	showVersion := flag.Bool("V", false, "バージョン")

	// /bin/sora-archive-uploader -C ./config.ini
	configFilePath := flag.String("C", "./config.ini", "Config file path")
	flag.Parse()

	if *showVersion {
		fmt.Printf("Sora Archive Uploader version %s\n", archive.Version)
		return
	}

	log.Printf("config file path: %s", *configFilePath)
	archive.Run(configFilePath)
}
