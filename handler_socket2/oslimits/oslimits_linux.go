package oslimits

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"syscall"
)

func SetOpenFilesLimit(num int) bool {

	is_linux := strings.Contains(strings.ToLower(runtime.GOOS), "linux")

	if !is_linux {
		return false
	}

	var rLimit syscall.Rlimit
	err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error Getting Rlimit "+err.Error())
		return false
	}

	rLimit.Max = uint64(num)
	rLimit.Cur = uint64(num)

	err = syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rLimit)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error Setting Rlimit "+err.Error())
		return false
	}

	err = syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error Getting Rlimit "+err.Error())
		return false
	}

	if rLimit.Max != uint64(num) || rLimit.Cur != uint64(num) {
		_tmp := fmt.Sprintf("Error Setting Rlimit, requested %d, got %d/%d ", num, rLimit.Cur, rLimit.Max)
		fmt.Fprintf(os.Stderr, _tmp)
	}

	return true
}
