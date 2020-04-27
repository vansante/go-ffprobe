package ffprobe

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
)

var (
	// ErrBinNotFound is returned when the ffprobe binary was not found
	ErrBinNotFound = errors.New("ffprobe bin not found")
	// ErrTimeout is returned when the ffprobe process did not succeed within the given time
	ErrTimeout = errors.New("ffprobe process timeout")

	binPath = "ffprobe"
)

// SetFFProbeBinPath sets the global path to find and execute the ffprobe program
func SetFFProbeBinPath(newBinPath string) {
	binPath = newBinPath
}

// ProbeURL is used for probing the given media file using ffprobe. The URL can be a local path, a HTTP URL or any other
// protocol supported by ffprobe, see here for a full list: https://ffmpeg.org/ffmpeg-protocols.html
// This function takes a context to allow killing the ffprobe process if it takes too long or in case of shutdown.
// Any additional ffprobe parameter can be supplied as well.
func ProbeURL(ctx context.Context, fileURL string, extraFFProbeOptions ...string) (data *ProbeData, err error) {
	args := append([]string{
		"-loglevel", "fatal",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
	}, extraFFProbeOptions...)

	// Add the file argument
	args = append(args, fileURL)

	cmd := exec.CommandContext(
		ctx,
		binPath,
		args...,
	)

	var outputBuf bytes.Buffer
	cmd.Stdout = &outputBuf

	err = cmd.Start()
	if errors.Is(err, exec.ErrNotFound) {
		return nil, ErrBinNotFound
	} else if err != nil {
		return nil, fmt.Errorf("error starting ffprobe: %w", err)
	}

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case <-ctx.Done():
		err = cmd.Process.Kill()
		if err == nil {
			return nil, ErrTimeout
		}
		return nil, err
	case err = <-done:
		if err != nil {
			return nil, err
		}
	}

	data = &ProbeData{}
	err = json.Unmarshal(outputBuf.Bytes(), data)
	if err != nil {
		return data, fmt.Errorf("error parsing ffprobe output: %w", err)
	}

	return data, nil
}
