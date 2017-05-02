package main

import (
	"encoding/json"
	"github.com/vansante/go-ffprobe"
	"log"
)

func main() {
	path := "D:/Downloads/big_buck_bunny.mp4"

	data, err := ffprobe.GetVideoData(path)
	if err != nil {
		log.Panicf("Error getting data: %v", err)
	}

	buf, err := json.MarshalIndent(data, "", "  ")
	log.Print(string(buf))

	buf, err = json.MarshalIndent(data.GetFirstVideoStream(), "", "  ")
	log.Print(string(buf))
}
