package main
import (
	"fmt"
	"net/http"
)
func main() {
	resp, err := http.Get("http://gluetun:8080/api/v2/torrents/info")
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
	} else {
		fmt.Printf("STATUS: %d\n", resp.StatusCode)
	}
}
