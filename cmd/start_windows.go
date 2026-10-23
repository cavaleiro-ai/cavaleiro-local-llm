package cmd

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"

	"github.com/ollama/ollama/api"
	"golang.org/x/sys/windows"
)

const (
	Installer = "OllamaSetup.exe"
)

func startApp(ctx context.Context, client *api.Client) error {
	if len(isProcRunning(Installer)) > 0 {
		return fmt.Errorf("upgrade in progress...")
	}
	AppName := "ollama app.exe"
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	appExe := filepath.Join(filepath.Dir(exe), AppName)
	_, err = os.Stat(appExe)
	if errors.Is(err, os.ErrNotExist) {
		// Try the standard install location
		localAppData := os.Getenv("LOCALAPPDATA")
		appExe = filepath.Join(localAppData, "Ollama", AppName)
		_, err := os.Stat(appExe)
		if errors.Is(err, os.ErrNotExist) {
			// Finally look in the path
			appExe, err = exec.LookPath(AppName)
			if err != nil {
				return errors.New("could not locate ollama app")
			}
		}
	}

	cmd_path := "c:\\Windows\\system32\\cmd.exe"
	cmd := exec.Command(cmd_path, "/c", appExe, "--hide", "--fast-startup")
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x08000000, HideWindow: true}

	cmd.Stdin = strings.NewReader("")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("unable to start ollama app %w", err)
	}

	if cmd.Process != nil {
		defer cmd.Process.Release() //nolint:errcheck
	}
	return waitForServer(ctx, client)
}

func isProcRunning(procName string) []uint32 {
	// A Toolhelp snapshot lists every process with its executable name
	// without opening each one. Opening every process and enumerating its
	// modules took seconds (minutes under load) on a machine with ~700
	// processes, and it misread the EnumProcesses byte count as a number of
	// process IDs.
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		slog.Debug("failed to check for running installers", "error", err)
		return nil
	}
	defer windows.CloseHandle(snapshot) //nolint:errcheck

	var matches []uint32
	entry := windows.ProcessEntry32{Size: uint32(unsafe.Sizeof(windows.ProcessEntry32{}))}
	for err = windows.Process32First(snapshot, &entry); err == nil; err = windows.Process32Next(snapshot, &entry) {
		if strings.EqualFold(windows.UTF16ToString(entry.ExeFile[:]), procName) {
			matches = append(matches, entry.ProcessID)
		}
	}
	if !errors.Is(err, windows.ERROR_NO_MORE_FILES) {
		slog.Debug("failed to check for running installers", "error", err)
	}
	return matches
}
