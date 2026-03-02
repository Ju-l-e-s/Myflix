package main

import (
	"fmt"
	"net/http"
	"net/url"
	"time"
	"io"
)

func main() {
	searchURL := "https://api.themoviedb.org/3/search/multi?query=" + url.QueryEscape("Titanic") + "&language=fr-FR"
	req, _ := http.NewRequest("GET", searchURL, nil)
	req.Header.Add("Authorization", "Bearer eyJhbGciOiJIUzI1NiJ9.eyJhdWQiOiI4MjFhY2IxMDU4YzdjYTkwYWJkZDZhNDY4OGZhNzRmNSIsIm5iZiI6MTc3MjEwMjY0MS41MDEsInN1YiI6IjY5YTAyM2YxYmZkOTZiMzE5NmU5M2ZhMCIsInNjb3BlcyI6WyJhcGlfcmVhZCJdLCJ2ZXJzaW9uIjoxfQ.lM6Gl1ckJuGOq_rMT1N-wQDuJu54GjqkVQouN2ULHCA")
	req.Header.Add("Accept", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("ERR: %v\n", err)
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("STATUS: %d\nBODY: %s\n", resp.StatusCode, string(body))
}
