package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetDirSize(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "dirsize_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a file with 100 bytes
	file1 := filepath.Join(tempDir, "file1.txt")
	content := make([]byte, 100)
	os.WriteFile(file1, content, 0644)

	// Create a subdirectory with a file with 200 bytes
	subDir := filepath.Join(tempDir, "sub")
	os.Mkdir(subDir, 0755)
	file2 := filepath.Join(subDir, "file2.txt")
	content2 := make([]byte, 200)
	os.WriteFile(file2, content2, 0644)

	size, err := getDirSize(tempDir)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if size != 300 {
		t.Errorf("Expected dir size to be 300, got %d", size)
	}
}

func TestGetEnvInt64(t *testing.T) {
	// Test with existing valid env var
	os.Setenv("TEST_INT64_ENV", "12345")
	defer os.Unsetenv("TEST_INT64_ENV")

	val := getEnvInt64("TEST_INT64_ENV", 999)
	if val != 12345 {
		t.Errorf("Expected 12345, got %d", val)
	}

	// Test with missing env var (should fallback)
	val = getEnvInt64("MISSING_TEST_INT64_ENV", 999)
	if val != 999 {
		t.Errorf("Expected fallback 999, got %d", val)
	}

	// Test with invalid env var (should fallback)
	os.Setenv("TEST_INVALID_INT64_ENV", "not_a_number")
	defer os.Unsetenv("TEST_INVALID_INT64_ENV")
	val = getEnvInt64("TEST_INVALID_INT64_ENV", 888)
	if val != 888 {
		t.Errorf("Expected fallback 888, got %d", val)
	}
}
