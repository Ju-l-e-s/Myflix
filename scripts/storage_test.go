package main

import (
	"testing"
)

func TestCleanTitle(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Inception.2010.1080p.BluRay.x264", "Inception 2010"},
		{"The.Revenant.2015.4k.UHD.HDR.x265", "The Revenant 2015"},
		{"The.Boys.S01E01.720p.WEB-DL-NTb", "The Boys S01E01"},
		{"Movie.Title.2023.1080p.AMZN.WEB-DL.DDP5.1.H.264", "Movie Title 2023"},
	}

	for _, tc := range tests {
		got := cleanTitle(tc.input)
		if got != tc.expected {
			t.Errorf("cleanTitle(%q) = %q; attendu %q", tc.input, got, tc.expected)
		}
	}
}

func TestGetNumberEmoji(t *testing.T) {
	if getNumberEmoji(1) != "1️⃣" {
		t.Errorf("Attendu 1️⃣, obtenu %s", getNumberEmoji(1))
	}
	if getNumberEmoji(10) != "🔟" {
		t.Errorf("Attendu 🔟, obtenu %s", getNumberEmoji(10))
	}
	if getNumberEmoji(11) != "11." {
		t.Errorf("Attendu 11., obtenu %s", getNumberEmoji(11))
	}
}
