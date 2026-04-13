package version

import (
	"testing"
)

func TestGet(t *testing.T) {
	info := Get()

	if info.Version == "" {
		t.Log("Version is empty (expected in dev build)")
	}
	if info.GoVersion == "" {
		t.Error("GoVersion should not be empty")
	}
	if info.OS == "" {
		t.Error("OS should not be empty")
	}
	if info.Arch == "" {
		t.Error("Arch should not be empty")
	}
}

func TestString(t *testing.T) {
	s := String()
	if s == "" {
		t.Error("String() should not return empty")
	}
}

func TestPrint(t *testing.T) {
	Print()
}

func TestIsDevelopment(t *testing.T) {
	result := IsDevelopment()
	if result {
		t.Log("Running in development mode")
	}
}

func TestIsRelease(t *testing.T) {
	result := IsRelease()
	if result {
		t.Log("Running in release mode")
	}
}

func TestVersionInfo(t *testing.T) {
	info := Info{
		Version:   "1.0.0",
		Commit:    "abc123",
		Date:      "2024-01-01",
		GoVersion: "go1.21",
		OS:        "linux",
		Arch:      "amd64",
	}

	if info.Version != "1.0.0" {
		t.Errorf("Expected Version '1.0.0', got '%s'", info.Version)
	}
	if info.Commit != "abc123" {
		t.Errorf("Expected Commit 'abc123', got '%s'", info.Commit)
	}
}

func TestVersionConstants(t *testing.T) {
	if Version != "" {
		t.Logf("Version = %s", Version)
	}
	if Commit != "" {
		t.Logf("Commit = %s", Commit)
	}
	if Date != "" {
		t.Logf("Date = %s", Date)
	}
}