package main

import (
	"bytes"
	"errors"
	"os/exec"
	"strings"
	"testing"
)

// "bestfit -check" on invalid UTF-8 printed "U+FFFD refused" and exited 0,
// while "bestfit cp932" on the same bytes says "invalid UTF-8" and exits 1.
// -check lists what would fail, so it must name the bad byte and fail too.
func TestRegress20261015_CheckInvalidUTF8Exits1(t *testing.T) {
	bin := build(t)
	cmd := exec.Command(bin, "-check", "cp932")
	cmd.Stdin = strings.NewReader("a\xffb〜")
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	var ee *exec.ExitError
	if err == nil || !errors.As(err, &ee) || ee.ExitCode() != 1 {
		t.Errorf("exit = %v, want status 1", err)
	}
	out := stdout.String()
	if !strings.Contains(out, "byte 1: invalid UTF-8") || strings.Contains(out, "FFFD") {
		t.Errorf("stdout = %q, want the invalid byte named and no U+FFFD line", out)
	}
	if !strings.Contains(out, "byte 3: 〜 U+301C -> ～") {
		t.Errorf("stdout = %q, want the later replacement still listed", out)
	}
}
