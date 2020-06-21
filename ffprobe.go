package ffprobe

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
)

var (
	binPath = "ffprobe"
)

// SetFFProbeBinPath sets the global path to find and execute the ffprobe program
func SetFFProbeBinPath(newBinPath string) {
	binPath = newBinPath
}

// ProbeURL is used to probe the given media file using ffprobe. The URL can be a local path, a HTTP URL or any other
// protocol supported by ffprobe, see here for a full list: https://ffmpeg.org/ffmpeg-protocols.html
// This function takes a context to allow killing the ffprobe process if it takes too long or in case of shutdown.
// Any additional ffprobe parameter can be supplied as well using extraFFProbeOptions.
func ProbeURL(ctx context.Context, fileURL string, extraFFProbeOptions ...string) (data *ProbeData, err error) {
	args := append([]string{
		"-loglevel", "fatal",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
	}, extraFFProbeOptions...)

	// Add the file argument
	args = append(args, fileURL)

	cmd := exec.CommandContext(ctx, binPath, args...)
	cmd.SysProcAttr = procAttributes()

	return runProbe(cmd)
}

// ProbeReader is used to probe a media file using an io.Reader. The reader is piped to the stdin of the ffprobe command
// and the data is returned.
// This function takes a context to allow killing the ffprobe process if it takes too long or in case of shutdown.
// Any additional ffprobe parameter can be supplied as well using extraFFProbeOptions.
func ProbeReader(ctx context.Context, reader io.Reader, extraFFProbeOptions ...string) (data *ProbeData, err error) {
	args := append([]string{
		"-loglevel", "fatal",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
	}, extraFFProbeOptions...)

	// Add the file from stdin argument
	args = append(args, "-")

	cmd := exec.CommandContext(ctx, binPath, args...)
	cmd.Stdin = reader
	cmd.SysProcAttr = procAttributes()

	return runProbe(cmd)
}

// runProbe takes the fully configured ffprobe command and executes it, returning the ffprobe data if everything went fine.
func runProbe(cmd *exec.Cmd) (data *ProbeData, err error) {
	var outputBuf bytes.Buffer
	var stdErr bytes.Buffer

	cmd.Stdout = &outputBuf
	cmd.Stderr = &stdErr

	err = cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("error running %s [%s] %w", binPath, stdErr.String(), err)
	}

	if stdErr.Len() > 0 {
		return nil, fmt.Errorf("ffprobe error: %s", stdErr.String())
	}

	data = &ProbeData{}
	err = json.Unmarshal(outputBuf.Bytes(), data)
	if err != nil {
		return data, fmt.Errorf("error parsing ffprobe output: %w", err)
	}

	return data, nil
}
