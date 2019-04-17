package main

import (
	"context"
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

	ctx, _ := context.WithTimeout(context.Background(), 500*time.Millisecond)

	data, err := ffprobe.GetProbeData(path, ctx)
	if err != nil {
		log.Panicf("Error getting data: %v", err)
	}

	buf, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		log.Panicf("Error unmarshalling: %v", err)
	}
	log.Print(string(buf))

	buf, err = json.MarshalIndent(data.GetFirstVideoStream(), "", "  ")
	if err != nil {
		log.Panicf("Error unmarshalling: %v", err)
	}
	log.Print(string(buf))

	log.Printf("%v", data.GetStreams(ffprobe.StreamVideo))

	log.Printf("\nDuration: %v\n", data.Format.Duration())
	log.Printf("\nStartTime: %v\n", data.Format.StartTime())
}
