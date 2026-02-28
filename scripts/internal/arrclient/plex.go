package arrclient

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"time"

	"myflixbot.local/internal/config"
)

type PlexClient struct {
	cfg    *config.Config
	client *http.Client
}

type PlexMetadata struct {
	Title      string `xml:"title,attr"`
	LastViewed int64  `xml:"lastViewedAt,attr"`
	AddedAt    int64  `xml:"addedAt,attr"`
	Size       int64  `xml:"size,attr"`
	Type       string `xml:"type,attr"`
}

type PlexLibrary struct {
	Metadata []PlexMetadata `xml:"Metadata"`
}

func NewPlexClient(cfg *config.Config, client *http.Client) *PlexClient {
	return &PlexClient{cfg: cfg, client: client}
}

// GetColdMedia récupère les médias non vus depuis plus de X mois
func (p *PlexClient) GetColdMedia(months int) ([]PlexMetadata, error) {
	// 1. Récupérer toutes les sections (Movies/TV)
	// Note: Pour simplifier, on itère sur les bibliothèques standard
	url := fmt.Sprintf("%s/library/sections/all/all?X-Plex-Token=%s", p.cfg.PlexURL, p.cfg.PlexToken)
	
	resp, err := p.client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var lib PlexLibrary
	if err := xml.NewDecoder(resp.Body).Decode(&lib); err != nil {
		return nil, err
	}

	var cold []PlexMetadata
	threshold := time.Now().AddDate(0, -months, 0).Unix()

	for _, m := range lib.Metadata {
		// Si le média a été vu, on vérifie la date. S'il n'a jamais été vu, on vérifie la date d'ajout.
		lastActivity := m.LastViewed
		if lastActivity == 0 {
			lastActivity = m.AddedAt
		}

		if lastActivity < threshold {
			cold = append(cold, m)
		}
	}

	return cold, nil
}
