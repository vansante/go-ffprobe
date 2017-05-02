package ffprobe

import (
	"encoding/json"
	"os/exec"
)

var binPath string = "ffprobe"

func SetFFProbeBinPath(newBinPath string) {
	binPath = newBinPath
}

func GetVideoData(filePath string) (data *ProbeData, err error) {
	cmd := exec.Command(
		binPath,
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		filePath,
	)

	outputBuf, err := cmd.Output()
	if err != nil {
		return
	}

	data = &ProbeData{}
	err = json.Unmarshal(outputBuf, data)
	if err != nil {
		return
	}
	return
}
