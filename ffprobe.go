  package ffprobe

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"time"
)

var (
	// ErrBinNotFound is returned when the ffprobe binary was not found
	ErrBinNotFound = errors.New("ffprobe bin not found")
	// ErrTimeout is returned when the ffprobe process did not succeed within the given time
	ErrTimeout = errors.New("process timeout exceeded")

	binPath = "ffprobe"
)

// SetFFProbeBinPath sets the global path to find and execute the ffprobe program
func SetFFProbeBinPath(newBinPath string) {
	binPath = newBinPath
}

// GetProbeData is used for probing the given media file using ffprobe with a set timeout.
// The timeout can be provided to kill the process if it takes too long to determine
// the files information.
// Note: It is probably better to use Context with GetProbeDataContext() these days as it is more flexible.
func GetProbeData(r io.Reader, timeout time.Duration) (data *ProbeData, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return GetProbeDataContext(ctx, r)
}

// GetProbeDataContext is used for probing the given media file using ffprobe.
// It takes a context to allow killing the ffprobe process if it takes too long or in case of shutdown.
func GetProbeDataContext(ctx context.Context, r io.Reader) (data *ProbeData, err error) {
	cmd := exec.Command(
		binPath,
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		"-",
	)
	cmd.Stdin = r

	var outputBuf bytes.Buffer
	cmd.Stdout = &outputBuf

	err = cmd.Start()
	if err == exec.ErrNotFound {
		return nil, ErrBinNotFound
	} else if err != nil {
		return nil, err
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
		return data, err
	}

	return data, nil
}

// GetProbeDataOptions is used for probing the given media file using ffprobe, optionally taking in extra arguments for ffprobe.
// It takes a context to allow killing the ffprobe process if it takes too long or in case of shutdown.
func GetProbeDataOptions(ctx context.Context, r io.Reader, extraFFProbeOptions ...string) (data *ProbeData, err error) {
	args := append([]string{
		"-loglevel", "fatal",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
	}, extraFFProbeOptions...)

	cmd := exec.CommandContext(
		ctx,
		binPath,
		args...,
	)
	cmd.Stdin = r

	var outputBuf bytes.Buffer
	var stdErr bytes.Buffer

	cmd.Stdout = &outputBuf
	cmd.Stderr = &stdErr

	err = cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("error running ffprobe [%s] %w", stdErr.String(), err)
	}

	if stdErr.String() != "" {
		return nil, fmt.Errorf("ffprobe error: %s", stdErr.String())
	}

	data = &ProbeData{}
	err = json.Unmarshal(outputBuf.Bytes(), data)
	if err != nil {
		return data, fmt.Errorf("error unmarshalling output: %w", err)
	}

	return data, nil
}
