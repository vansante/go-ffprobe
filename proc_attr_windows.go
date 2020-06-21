// +build windows

package ffprobe

import (
	"syscall"
)

func procAttributes() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		HideWindow: true,
	}
}
