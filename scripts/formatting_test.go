package main

import (
	"testing"
)

func TestGetStorageEmoji(t *testing.T) {
	tests := []struct {
		path      string
		expected  string
	}{
		{"/mnt/externe/movies/test", "📚"},
		{"/data/internal/movies/test", "🚀"},
		{"/movies/test", "🚀"},
	}

	for _, tt := range tests {
		result := getStorageEmoji(tt.path)
		if result != tt.expected {
			t.Errorf("getStorageEmoji(%q) = %s, want %s", tt.path, result, tt.expected)
		}
	}
}

func TestGetIndexEmoji(t *testing.T) {
	tests := []struct {
		index    int
		expected string
	}{
		{1, "1️⃣"},
		{5, "5️⃣"},
		{10, "🔟"},
		{15, "15."}, // Fallback behavior prints the number followed by a dot
		{0, "0."},
	}

	for _, tt := range tests {
		result := getIndexEmoji(tt.index)
		if result != tt.expected {
			t.Errorf("getIndexEmoji(%d) = %s, want %s", tt.index, result, tt.expected)
		}
	}
}

func TestFormatETA(t *testing.T) {
	tests := []struct {
		seconds  int
		expected string
	}{
		{8640000, "calcul..."},
		{0, "0:00:00"},
		{45, "0:00:45"},
		{120, "0:02:00"},
		{3660, "1:01:00"},
		{90060, "1 days, 1:01:00"},
	}

	for _, tt := range tests {
		result := formatETA(tt.seconds)
		if result != tt.expected {
			t.Errorf("formatETA(%d) = %s, want %s", tt.seconds, result, tt.expected)
		}
	}
}
