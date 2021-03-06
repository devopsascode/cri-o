package config

import (
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
	"syscall"

	"golang.org/x/sys/unix"
)

// isCgroup2UnifiedMode returns whether we are running in cgroup2 mode.
func isCgroup2UnifiedMode() (isUnified bool, isUnifiedErr error) {
	cgroupRoot := "/sys/fs/cgroup"

	var st syscall.Statfs_t
	if err := syscall.Statfs(cgroupRoot, &st); err != nil {
		isUnified, isUnifiedErr = false, err
	} else {
		isUnified, isUnifiedErr = st.Type == unix.CGROUP2_SUPER_MAGIC, nil
	}
	return
}

const (
	oldMaxSize = uint64(1048576)
)

// getDefaultProcessLimits returns the nofile and nproc for the current process in ulimits format
// Note that nfile sometimes cannot be set to unlimited, and the limit is hardcoded
// to (oldMaxSize) 1048576 (2^20), see: http://stackoverflow.com/a/1213069/1811501
// In rootless containers this will fail, and the process will just use its current limits
func getDefaultProcessLimits() []string {
	rlim := unix.Rlimit{Cur: oldMaxSize, Max: oldMaxSize}
	oldrlim := rlim
	// Attempt to set file limit and process limit to pid_max in OS
	dat, err := ioutil.ReadFile("/proc/sys/kernel/pid_max")
	if err == nil {
		val := strings.TrimSuffix(string(dat), "\n")
		max, err := strconv.ParseUint(val, 10, 64)
		if err == nil {
			rlim = unix.Rlimit{Cur: uint64(max), Max: uint64(max)}
		}
	}
	defaultLimits := []string{}
	if err := unix.Setrlimit(unix.RLIMIT_NOFILE, &rlim); err == nil {
		defaultLimits = append(defaultLimits, fmt.Sprintf("nofile=%d:%d", rlim.Cur, rlim.Max))
	} else {
		if err := unix.Setrlimit(unix.RLIMIT_NOFILE, &oldrlim); err == nil {
			defaultLimits = append(defaultLimits, fmt.Sprintf("nofile=%d:%d", oldrlim.Cur, oldrlim.Max))
		}
	}
	if err := unix.Setrlimit(unix.RLIMIT_NPROC, &rlim); err == nil {
		defaultLimits = append(defaultLimits, fmt.Sprintf("nproc=%d:%d", rlim.Cur, rlim.Max))
	} else {
		if err := unix.Setrlimit(unix.RLIMIT_NPROC, &oldrlim); err == nil {
			defaultLimits = append(defaultLimits, fmt.Sprintf("nproc=%d:%d", oldrlim.Cur, oldrlim.Max))
		}
	}
	return defaultLimits
}
