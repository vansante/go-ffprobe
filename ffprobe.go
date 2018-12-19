package ffprobe

import (
	"bytes"
	"encoding/json"
	"errors"
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

// GetProbeData is the main command used for probing the given media file using ffprobe.
// A timeout can be provided to kill the process if it takes too long to determine
// the files information.
func GetProbeData(filePath string, timeout time.Duration) (data *ProbeData, err error) {
	cmd := exec.Command(
		binPath,
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		filePath,
	)

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
	case <-time.After(timeout):
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
