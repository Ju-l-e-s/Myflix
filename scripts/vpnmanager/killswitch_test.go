package vpnmanager

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestKillswitchIntegration(t *testing.T) {
	realIP := "82.10.10.10"
	var qbitPaused int32

	// 1. Mock qBittorrent Server
	qbitServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v2/torrents/pause" {
			atomic.StoreInt32(&qbitPaused, 1)
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer qbitServer.Close()

	// 2. Mock IP Provider (Simulating a Leak)
	ipServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(realIP))
	}))
	defer ipServer.Close()

	// 3. Initialize Manager with mocks
	mgr := NewManager(nil, 12345, realIP, qbitServer.URL, false, "")
	mgr.SetIPCheckerURL(ipServer.URL)

	// 4. Trigger IP Update (should detect leak)
	_, err := mgr.UpdateIP()
	if err != nil {
		t.Fatalf("UpdateIP failed: %v", err)
	}

	// 5. Verification
	if atomic.LoadInt32(&qbitPaused) != 1 {
		t.Error("❌ KILLSWITCH FAIL: qBittorrent was NOT paused during an IP leak simulation!")
	} else {
		t.Log("✅ KILLSWITCH SUCCESS: qBittorrent was correctly paused.")
	}
}
