package main

import (
	"strings"
	"testing"
)

func TestCreateStatusMsgStyle(t *testing.T) {
	// "Gold Standard" de l'interface de stockage
	used, total := 50.0, 100.0
	label, icon, tier := "NVMe", "🚀", "Hot Tier"
	
	msg := createStatusMsg(used, total, label, icon, tier)
	
	// Vérification de la structure et du style
	if !strings.Contains(msg, "🚀 NVMe (Hot Tier)") {
		t.Errorf("Style de l'en-tête de stockage modifié : %s", msg)
	}
	
	// Vérification de la barre de progression (18 caractères)
	expectedBar := "█████████░░░░░░░░░" // 50% de 18
	if !strings.Contains(msg, expectedBar) {
		t.Errorf("Style de la barre de progression stockage modifié. Attendu: %s", expectedBar)
	}
	
	// Vérification des emojis de statut
	if !strings.Contains(msg, "🟢 Libre") {
		t.Errorf("Emoji de statut 'OK' manquant ou modifié")
	}
}

func TestFormatMovieDetailsStyle(t *testing.T) {
	m := Movie{
		Title:      "Inception",
		Year:       2010,
		Runtime:    148,
		SizeOnDisk: 21474836480, // 20 GB
		Path:       "/data/media/movies/Inception (2010)",
		Director:   "Christopher Nolan",
	}
	
	msg := formatMovieDetails(m)
	
	// Vérification de l'en-tête
	if !strings.Contains(msg, "🎬 <b>Inception</b> (2010)") {
		t.Errorf("Format du titre du film modifié")
	}
	
	// Vérification du séparateur visuel
	separator := "⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯"
	if !strings.Contains(msg, separator) {
		t.Errorf("Séparateur visuel manquant ou modifié")
	}
	
	// Vérification des métadonnées
	if !strings.Contains(msg, "👤 <b>Réal.</b> : Christopher Nolan") {
		t.Errorf("Format du réalisateur modifié")
	}
	if !strings.Contains(msg, "⏱️ <b>Durée</b> : 2h 28m") {
		t.Errorf("Format de la durée modifié")
	}
	if !strings.Contains(msg, "⚖️ <b>Poids</b> : 20.00 GB") {
		t.Errorf("Format du poids modifié")
	}
}

func TestFormatSeriesDetailsStyle(t *testing.T) {
	s := Series{
		Title:   "The Boys",
		Year:    2019,
		Runtime: 60,
		Path:    "/data/media/tv/The Boys",
		Statistics: map[string]interface{}{
			"sizeOnDisk": 53687091200.0, // 50 GB
		},
	}
	
	msg := formatSeriesDetails(s)
	
	if !strings.Contains(msg, "📺 <b>The Boys</b> (2019)") {
		t.Errorf("Format du titre de la série modifié")
	}
	if !strings.Contains(msg, "⚖️ <b>Poids</b> : 50.00 GB") {
		t.Errorf("Format du poids de la série modifié")
	}
}

func TestGetProgressBar(t *testing.T) {
	// Test du style de la barre de téléchargement (15 caractères)
	bar := getProgressBar(66.6)
	expected := "█████████░░░░░░" // 9/15 pour ~66% (60% réel car int truncation)
	if bar != expected {
		t.Errorf("Style de la barre de téléchargement modifié. Attendu: %s, Reçu: %s", expected, bar)
	}
}

func TestFormatSpeed(t *testing.T) {
	if formatSpeed(1024*1024*5.5) != "5.5 MB/s" {
		t.Errorf("Format de vitesse MB/s modifié")
	}
	if formatSpeed(1024*512) != "512.0 KB/s" {
		t.Errorf("Format de vitesse KB/s modifié")
	}
}
