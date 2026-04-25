package service

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

func TestStartDarwinLoadsInstalledServiceWhenUnloaded(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("darwin-specific test")
	}

	tempDir := t.TempDir()
	launchctlLog := filepath.Join(tempDir, "launchctl.log")
	launchctlPath := filepath.Join(tempDir, "launchctl")
	script := "#!/bin/sh\n" +
		"printf '%s\\n' \"$*\" >> \"$LAUNCHCTL_LOG\"\n" +
		"case \"$1\" in\n" +
		"  list) exit 1 ;;\n" +
		"  load) exit 0 ;;\n" +
		"  start) exit 3 ;;\n" +
		"esac\n" +
		"exit 0\n"
	if err := os.WriteFile(launchctlPath, []byte(script), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	// #nosec G302 -- test stub must be executable to simulate launchctl.
	if err := os.Chmod(launchctlPath, 0o755); err != nil {
		t.Fatalf("Chmod() error = %v", err)
	}

	t.Setenv("PATH", tempDir+":"+os.Getenv("PATH"))
	t.Setenv("LAUNCHCTL_LOG", launchctlLog)

	plistPath := filepath.Join(tempDir, "Library", "LaunchAgents", "com.foxytanuki.rcode-server.plist")
	if err := os.MkdirAll(filepath.Dir(plistPath), 0o700); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(plistPath, []byte("plist"), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	sm := &ServiceManager{userHome: tempDir}

	if err := sm.startDarwin(); err != nil {
		t.Fatalf("startDarwin() error = %v", err)
	}

	logData, err := os.ReadFile(launchctlLog) // #nosec G304 -- launchctlLog is created inside this test's temp directory
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	got := string(logData)
	if got != "list com.foxytanuki.rcode-server\nload "+plistPath+"\n" {
		t.Fatalf("launchctl calls = %q", got)
	}
}

func TestGenerateDarwinPlistIncludesHomebrewPath(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("darwin-specific test")
	}

	sm := &ServiceManager{userHome: "/Users/tester"}
	plist := sm.generateDarwinPlist("/usr/local/bin/rcode-server")

	if !strings.Contains(plist, "/opt/homebrew/bin:/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin") {
		t.Fatalf("plist PATH missing Homebrew bin: %s", plist)
	}
}

func TestInstallDarwinReloadsServiceWithBootoutBootstrap(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("darwin-specific test")
	}

	tempDir := t.TempDir()
	launchctlLog := filepath.Join(tempDir, "launchctl.log")
	launchctlPath := filepath.Join(tempDir, "launchctl")
	script := "#!/bin/sh\n" +
		"printf '%s\\n' \"$*\" >> \"$LAUNCHCTL_LOG\"\n" +
		"case \"$1\" in\n" +
		"  bootout) exit 0 ;;\n" +
		"  bootstrap) exit 0 ;;\n" +
		"esac\n" +
		"exit 1\n"
	if err := os.WriteFile(launchctlPath, []byte(script), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	// #nosec G302 -- test stub must be executable to simulate launchctl.
	if err := os.Chmod(launchctlPath, 0o755); err != nil {
		t.Fatalf("Chmod() error = %v", err)
	}

	binaryPath := filepath.Join(tempDir, "rcode-server")
	if err := os.WriteFile(binaryPath, []byte("#!/bin/sh\nexit 0\n"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	// #nosec G302 -- test binary must be executable for installDarwin.
	if err := os.Chmod(binaryPath, 0o755); err != nil {
		t.Fatalf("Chmod() error = %v", err)
	}

	t.Setenv("PATH", tempDir+":"+os.Getenv("PATH"))
	t.Setenv("LAUNCHCTL_LOG", launchctlLog)

	sm := &ServiceManager{binaryPath: binaryPath, userHome: tempDir}

	if err := sm.installDarwin(); err != nil {
		t.Fatalf("installDarwin() error = %v", err)
	}

	plistPath := filepath.Join(tempDir, "Library", "LaunchAgents", "com.foxytanuki.rcode-server.plist")
	logData, err := os.ReadFile(launchctlLog) // #nosec G304 -- launchctlLog is created inside this test's temp directory
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	got := string(logData)
	uid := os.Getuid()
	want := "bootout gui/" + strconv.Itoa(uid) + " " + plistPath + "\nbootstrap gui/" + strconv.Itoa(uid) + " " + plistPath + "\n"
	if got != want {
		t.Fatalf("launchctl calls = %q, want %q", got, want)
	}
}
