package main

import (
	"encoding/json"
	"log"
	"os"
	"time"

	ffprobe "github.com/vansante/go-ffprobe"
)

func main() {
	if len(os.Args) < 2 {
		log.Println("Please provide the path to the file to analyze")
		os.Exit(1)
	}
	path := os.Args[1]

	data, err := ffprobe.GetProbeData(path, 500*time.Millisecond)
	if err != nil {
		log.Panicf("Error getting data: %v", err)
	}

	buf, err := json.MarshalIndent(data, "", "  ")
	log.Print(string(buf))

	buf, err = json.MarshalIndent(data.GetFirstVideoStream(), "", "  ")
	log.Print(string(buf))

	log.Printf("%v", data.GetStreams(ffprobe.StreamVideo))

	log.Printf("\nDuration: %v\n", data.Format.Duration())
	log.Printf("\nStartTime: %v\n", data.Format.StartTime())
}
