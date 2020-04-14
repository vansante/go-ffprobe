package ffprobe_test

import (
	"testing"
	"time"

	ffprobe "github.com/vansante/go-ffprobe"
)

func Test_ffprobe(t *testing.T) {
	// test GetProbeData
	path := "test.mp4"
	data, err := ffprobe.GetProbeData(path, 500*time.Millisecond)
	if err != nil {
		t.Errorf("Error getting data: %v", err)
	}

	// test ProbeData.GetStream
	stream := data.GetStreams(ffprobe.StreamVideo)
	if len(stream) != 1 {
		t.Errorf("wrong stream length.")
	}

	stream = data.GetStreams(ffprobe.StreamAudio)
	if len(stream) != 1 {
		t.Errorf("wrong stream length.")
	}

	// this stream is []
	data.GetStreams(ffprobe.StreamSubtitle)

	stream = data.GetStreams(ffprobe.StreamAny)
	if len(stream) != 2 {
		t.Errorf("wrong stream length.")
	}

	// test Format.Duration
	udration := data.Format.Duration()
	if udration.Seconds() != 5.312 {
		t.Errorf("this video is 5.312s.")
	}
	// test Format.StartTime
	startTime := data.Format.StartTime()
	if startTime != time.Duration(0) {
		t.Errorf("this video starts at 0s.")
	}
}
