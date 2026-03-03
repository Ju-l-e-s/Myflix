package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

func main() {
	apiKey := "1f875e27320201d7671e1d9a9f2d7983"
	url := "http://127.0.0.1:6767/api/system/settings"
	
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("X-Api-Key", apiKey)
	
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error GET:", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	
	var settings map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&settings); err != nil {
		fmt.Println("Error Decode:", err)
		os.Exit(1)
	}
	
	// Pipeline "Direct Play" 100%
	if general, ok := settings["general"].(map[string]interface{}); ok {
		fmt.Println("- Configuration Performance (2 threads) & Exclusion Images")
		general["concurrent_jobs"] = 2
		general["ignore_pgs_subs"] = true
		general["ignore_vobsub_subs"] = true
		general["embedded_subtitles_parser"] = "ffprobe"
	}
	
	if subsync, ok := settings["subsync"].(map[string]interface{}); ok {
		fmt.Println("- Activation Audio-to-Text (Subsync Reference Audio)")
		subsync["use_subsync"] = true
		subsync["force_audio"] = true 
	}

	payload, _ := json.Marshal(settings)
	reqPost, _ := http.NewRequest("POST", url, bytes.NewBuffer(payload))
	reqPost.Header.Set("X-Api-Key", apiKey)
	reqPost.Header.Set("Content-Type", "application/json")
	
	respPost, err := client.Do(reqPost)
	if err != nil {
		fmt.Println("Error POST:", err)
		os.Exit(1)
	}
	defer respPost.Body.Close()

	if respPost.StatusCode == 200 || respPost.StatusCode == 204 {
		fmt.Println("SUCCESS: Bazarr est maintenant verrouille.")
	} else {
		fmt.Println("ERROR: HTTP", respPost.StatusCode)
	}
}
