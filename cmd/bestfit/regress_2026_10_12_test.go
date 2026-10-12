package main

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// build compiles the command once per test binary and returns its path.
func build(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "bestfit")
	if out, err := exec.Command("go", "build", "-o", bin, ".").CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
	return bin
}

// A write to stdout that fails (full disk, closed descriptor) must not exit 0
// as if the bytes had been written. A read-only descriptor as stdout makes
// write(2) fail with EBADF deterministically, without SIGPIPE.
func TestRegress20261012_StdoutWriteErrorExits1(t *testing.T) {
	bin := build(t)
	ro, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}
	defer ro.Close()
	cmd := exec.Command(bin, "cp932")
	cmd.Stdin = strings.NewReader("東京〜大阪")
	cmd.Stdout = ro
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err = cmd.Run()
	var ee *exec.ExitError
	if err == nil || !errors.As(err, &ee) || ee.ExitCode() != 1 {
		t.Errorf("exit = %v (stderr %q), want status 1", err, stderr.String())
	}
	if !strings.Contains(stderr.String(), "bad file descriptor") {
		t.Errorf("stderr = %q, want the write error", stderr.String())
	}
}

// Trailing arguments are not flags once the encoding name has been seen, so
// "bestfit cp932 -check" used to convert instead of checking, writing Shift_JIS
// to the terminal. Anything but exactly one argument is a usage error.
func TestRegress20261012_ExtraArgIsUsageError(t *testing.T) {
	bin := build(t)
	cmd := exec.Command(bin, "cp932", "-check")
	cmd.Stdin = strings.NewReader("東京〜大阪")
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	var ee *exec.ExitError
	if err == nil || !errors.As(err, &ee) || ee.ExitCode() != 2 {
		t.Errorf("exit = %v, want status 2", err)
	}
	if stdout.Len() != 0 || !strings.Contains(stderr.String(), "usage:") {
		t.Errorf("stdout % X, stderr %q; want no output and usage", stdout.Bytes(), stderr.String())
	}
}
