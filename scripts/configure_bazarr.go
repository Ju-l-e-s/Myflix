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
	apiKey := "1f875e27320201d7671e1d9a9f2d7983"
	baseURL := "http://127.0.0.1:6767/api"

	// 1. Récupération des paramètres actuels
	req, _ := http.NewRequest("GET", baseURL+"/system/settings", nil)
	req.Header.Set("X-Api-Key", apiKey)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Erreur connexion GET: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	var settings map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&settings); err != nil {
		fmt.Printf("Erreur décodage JSON: %v\n", err)
		os.Exit(1)
	}

	// 2. Modification des paramètres
	// Section Subsync (Subtitles)
	if subsync, ok := settings["subsync"].(map[string]interface{}); ok {
		subsync["use_subsync"] = true
		subsync["force_audio"] = true
		subsync["max_offset_seconds"] = 60
	}

	// Section General (Concurrent Jobs)
	if general, ok := settings["general"].(map[string]interface{}); ok {
		general["concurrent_jobs"] = 4
	}

	// 3. Envoi des nouveaux paramètres
	payload, _ := json.Marshal(settings)
	reqPut, _ := http.NewRequest("PUT", baseURL+"/system/settings", bytes.NewBuffer(payload))
	reqPut.Header.Set("X-Api-Key", apiKey)
	reqPut.Header.Set("Content-Type", "application/json")
	reqPut.Header.Set("Accept", "application/json")

	respPut, err := client.Do(reqPut)
	if err != nil {
		fmt.Printf("Erreur connexion PUT: %v\n", err)
		os.Exit(1)
	}
	defer respPut.Body.Close()

	if respPut.StatusCode == 200 || respPut.StatusCode == 204 {
		fmt.Println("SUCCESS: Configuration mise à jour")
	} else {
		body, _ := io.ReadAll(respPut.Body)
		fmt.Printf("ERROR HTTP %d: %s\n", respPut.StatusCode, string(body))
	}
}
