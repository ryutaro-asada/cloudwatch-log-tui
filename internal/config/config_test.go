package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInitLoggingCreatesPrivateFile(t *testing.T) {
	c := &Config{LogFile: filepath.Join(t.TempDir(), "sub", "app.log")}
	f, err := c.InitLogging()
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	info, err := os.Stat(c.LogFile)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("log file mode = %o, want 600", perm)
	}
}

func TestNewUsesAbsolutePath(t *testing.T) {
	if p := New().LogFile; !filepath.IsAbs(p) || filepath.Base(p) != "cloudwatch-log-tui.log" {
		t.Errorf("LogFile = %q", p)
	}
}
