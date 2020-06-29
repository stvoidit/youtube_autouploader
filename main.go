package main

import (
	"log"
	"os"
	"path/filepath"

	"local.project/youtube_uploader/ytuploader"
)

func init() {
	var pwd, _ = os.Getwd()
	var dataFolder = filepath.Join(pwd, "data")
	if err := os.MkdirAll(dataFolder, os.ModePerm); err != nil {
		panic(err)
	}
}

func main() {
	startWather()
}

func startWather() {
	log.Println("Start")
	var c = ytuploader.NewClient()
	var w = ytuploader.NewWatcher("config.yaml")
	for f := range w.Watch() {
		log.Println("Новый файл:", f.Fullpath())
		if err := c.UploadVideo(f.Fullpath()); err != nil {
			log.Println("ERROR UPLOAD:", err)
		}
	}
}
