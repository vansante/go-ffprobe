package ffprobe

import (
	"encoding/json"
	"errors"
	"os/exec"
	"time"
	"bytes"
)

var ErrBinNotFound error = errors.New("ffprobe bin not found")
var ErrTimeout error = errors.New("process timeout exceeded")

var binPath string = "ffprobe"

func SetFFProbeBinPath(newBinPath string) {
	binPath = newBinPath
}

func GetVideoData(filePath string, timeout time.Duration) (data *ProbeData, err error) {
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
	if err != nil {
		if err == exec.ErrNotFound {
			err = ErrBinNotFound
		}
		return
	}

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case <-time.After(timeout):
		err = cmd.Process.Kill()
		if err == nil {
			err = ErrTimeout
		}
		return
	case err = <-done:
		if err != nil {
			return
		}
	}

	data = &ProbeData{}
	err = json.Unmarshal(outputBuf.Bytes(), data)
	if err != nil {
		return
	}
	return
}
