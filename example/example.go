package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"time"

	"gopkg.in/vansante/go-ffprobe.v2"
)

func main() {
	if len(os.Args) < 2 {
		log.Println("Please provide the path to the file to analyze")
		os.Exit(1)
	}
	path := os.Args[1]

	ctx, cancelFn := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelFn()

	data, err := ffprobe.ProbeURL(ctx, path)
	if err != nil {
		log.Panicf("Error getting data: %v", err)
	}

	buf, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		log.Panicf("Error unmarshalling: %v", err)
	}
	log.Print(string(buf))

	buf, err = json.MarshalIndent(data.FirstVideoStream(), "", "  ")
	if err != nil {
		log.Panicf("Error unmarshalling: %v", err)
	}
	log.Print(string(buf))

	log.Printf("%v", data.StreamType(ffprobe.StreamVideo))

	log.Printf("\nDuration: %v\n", data.Format.Duration())
	log.Printf("\nStartTime: %v\n", data.Format.StartTime())
}
