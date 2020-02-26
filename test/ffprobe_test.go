package ffprobe_test

import (
	"encoding/json"
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

	// test data
	_, err = json.MarshalIndent(data, "", "  ")
	if err != nil {
		t.Errorf("Error unmarshalling: %v", err)
	}

    // test ProbeData.GetFirestVideoStream
    _, err = json.MarshalIndent(data.GetFirstVideoStream(), "", "  ")
	if err != nil {
		t.Errorf("Error unmarshalling: %v", err)
	}

    // test ProbeData.GetFirstAudioStream
	_, err = json.MarshalIndent(data.GetFirstAudioStream(), "", "  ")
	if err != nil {
		t.Errorf("Error unmarshalling: %v", err)
	}

	// test ProbeData.GetFirstSubtitleStream
	_, err = json.MarshalIndent(data.GetFirstSubtitleStream(), "", "  ")
	if err != nil {
		t.Errorf("Error unmarshalling: %v", err)
	}

    // test ProbeData.GetStream
    stream := data.GetStreams(ffprobe.StreamVideo)
    if len(stream) == 0 {
    	t.Errorf("don't get stream.")
    }

    stream = data.GetStreams(ffprobe.StreamAudio)
    if len(stream) == 0 {
    	t.Errorf("don't get stream.")
    }

    // this stream is []
    stream = data.GetStreams(ffprobe.StreamSubtitle)

    stream = data.GetStreams(ffprobe.StreamAny)
    if len(stream) == 0 {
    	t.Errorf("don't get stream.")
    }

    // test Format.Duration
    udration := data.Format.Duration()
    if udration < time.Duration(60) {
    	t.Errorf("this video is more than 60s.")
    }
    // test Format.StartTime
    startTime := data.Format.StartTime()
    if startTime != time.Duration(0) {
    	t.Errorf("this video starts at 0s.")
    }
}
