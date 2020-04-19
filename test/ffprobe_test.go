package ffprobe

import (
	"testing"
	"time"

	ffprobe "github.com/vansante/go-ffprobe"
)

func Test_ffprobe(t *testing.T) {
	// test GetProbeData
	path := "test.mp4"
	data, err := ffprobe.GetProbeData(path, time.Second)
	if err != nil {
		t.Errorf("Error getting data: %v", err)
	}

	// test ProbeData.GetStream
	stream := data.GetStreams(ffprobe.StreamVideo)
	if len(stream) != 1 {
		t.Errorf("It just has one video stream.")
	}

	stream = data.GetStreams(ffprobe.StreamAudio)
	if len(stream) != 1 {
		t.Errorf("It just has one audio stream.")
	}

	// this stream is []
	stream = data.GetStreams(ffprobe.StreamSubtitle)
	if len(stream) != 0 {
		t.Errorf("it does not have subtitle stream.")
	}

	stream = data.GetStreams(ffprobe.StreamAny)
	if len(stream) != 2 {
		t.Errorf("It should have two streams.")
	}

	// test Format.Duration
	dration := data.Format.Duration()
	if dration.Seconds() != 5.312 {
		t.Errorf("this video is 5.312s.")
	}
	// test Format.StartTime
	startTime := data.Format.StartTime()
	if startTime != time.Duration(0) {
		t.Errorf("this video starts at 0s.")
	}
}
