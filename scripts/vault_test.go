package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestIsFileModified(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "vault_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	sourcePath := filepath.Join(tempDir, "source.txt")
	targetPath := filepath.Join(tempDir, "target.txt")

	// Test 1: Target does not exist (should return true)
	os.WriteFile(sourcePath, []byte("test"), 0644)
	modified, err := isFileModified(sourcePath, targetPath)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if !modified {
		t.Error("Expected modified to be true when target does not exist")
	}

	// Test 2: Target exists but is older (should return true)
	os.WriteFile(targetPath, []byte("test"), 0644)
	// Force target to be older
	oldTime := time.Now().Add(-1 * time.Hour)
	os.Chtimes(targetPath, oldTime, oldTime)
	
	modified, err = isFileModified(sourcePath, targetPath)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if !modified {
		t.Error("Expected modified to be true when source is newer")
	}

	// Test 3: Target is newer (should return false)
	newTime := time.Now().Add(1 * time.Hour)
	os.Chtimes(targetPath, newTime, newTime)
	
	modified, err = isFileModified(sourcePath, targetPath)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if modified {
		t.Error("Expected modified to be false when target is newer")
	}

	// Test 4: Source does not exist (should return error)
	os.Remove(sourcePath)
	_, err = isFileModified(sourcePath, targetPath)
	if err == nil {
		t.Error("Expected error when source does not exist")
	}
}
