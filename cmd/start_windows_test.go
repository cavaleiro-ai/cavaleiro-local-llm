package cmd

import "testing"

func TestIsProcRunningFindsCurrentProcess(t *testing.T) {
	// The test binary itself is running, so it must be found by its name.
	if len(isProcRunning("cmd.test.exe")) == 0 {
		t.Fatal("cmd.test.exe is not reported as running")
	}
}
