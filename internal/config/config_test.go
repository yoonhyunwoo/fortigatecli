package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSaveAndLoad(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	cfg := Config{
		Host:     "https://fortigate.example.com",
		Token:    "secret-token",
		VDOM:     "",
		Insecure: true,
		Timeout:  0,
	}

	if err := Save(cfg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	path, err := Path()
	if err != nil {
		t.Fatalf("Path() error = %v", err)
	}

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("config file not written: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if loaded.VDOM != "root" {
		t.Fatalf("VDOM default = %q, want root", loaded.VDOM)
	}
	if loaded.Timeout != 10*time.Second {
		t.Fatalf("Timeout default = %s, want 10s", loaded.Timeout)
	}
}

func TestPathUsesHomeDirectory(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	got, err := Path()
	if err != nil {
		t.Fatalf("Path() error = %v", err)
	}

	want := filepath.Join(home, ".fortigatecli", "config.yaml")
	if got != want {
		t.Fatalf("Path() = %q, want %q", got, want)
	}
}
