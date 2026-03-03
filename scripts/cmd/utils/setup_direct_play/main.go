package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

func main() {
	apiKey := os.Getenv("BAZARR_API_KEY")
	if apiKey == "" {
		fmt.Println("Erreur: BAZARR_API_KEY non définie")
		os.Exit(1)
	}
	baseURL := "http://127.0.0.1:6767/api"

	// 1. Get settings
	req, _ := http.NewRequest("GET", baseURL+"/system/settings", nil)
	req.Header.Set("X-Api-Key", apiKey)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	var settings map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&settings)

	// 2. Enforce Direct Play settings in Bazarr
	if general, ok := settings["general"].(map[string]interface{}); ok {
		fmt.Println("Enforcing Bazarr 'Direct Play' settings (ignoring image subtitles)...")
		general["ignore_pgs_subs"] = true
		general["ignore_vobsub_subs"] = true
		general["ignore_ass_subs"] = false
		general["embedded_subtitles_parser"] = "ffprobe"
	}

	// 3. Update settings
	payload, _ := json.Marshal(settings)
	reqPost, _ := http.NewRequest("POST", baseURL+"/system/settings", bytes.NewBuffer(payload))
	reqPost.Header.Set("X-Api-Key", apiKey)
	reqPost.Header.Set("Content-Type", "application/json")

	respPost, err := client.Do(reqPost)
	if err != nil {
		fmt.Printf("Error updating settings: %v\n", err)
		os.Exit(1)
	}
	defer respPost.Body.Close()

	if respPost.StatusCode == 200 || respPost.StatusCode == 204 {
		fmt.Println("SUCCESS: Bazarr settings updated.")
	} else {
		body, _ := io.ReadAll(respPost.Body)
		fmt.Printf("ERROR HTTP %d: %s\n", respPost.StatusCode, string(body))
	}
}
