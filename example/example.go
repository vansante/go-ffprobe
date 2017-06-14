package main

import (
	"encoding/json"
	"github.com/vansante/go-ffprobe"
	"log"
	"time"
)

func main() {
	path := "D:/Downloads/videorientation.mp4"

	data, err := ffprobe.GetVideoData(path, 500 * time.Millisecond)
	if err != nil {
		log.Panicf("Error getting data: %v", err)
	}

	buf, err := json.MarshalIndent(data, "", "  ")
	log.Print(string(buf))

	buf, err = json.MarshalIndent(data.GetFirstVideoStream(), "", "  ")
	log.Print(string(buf))

	log.Printf("\nDuration: %v\n", data.Format.Duration())
	log.Printf("\nStartTime: %v\n", data.Format.StartTime())

	//start := time.Now()
	//for i := 0; i < 100; i++ {
	//	_, err = ffprobe.GetVideoData(path)
	//	if err != nil {
	//		log.Panicf("Error getting data: %v", err)
	//	}
	//}
	//log.Printf("100 times get time: %v", time.Now().Sub(start))
}
