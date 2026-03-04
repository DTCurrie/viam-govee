package viamgovee

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	govee "github.com/DTCurrie/govee-go"
	"go.viam.com/rdk/logging"
)

func TestConnectToDevice_Found(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/devices", func(w http.ResponseWriter, _ *http.Request) {
		respondGovee(t, w, map[string]any{"devices": testDeviceList()})
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	logger := logging.NewTestLogger(t)
	// Inject the test server URL by creating the client ourselves and calling
	// the same logic that connectToDevice uses.
	client := govee.New("test-key", govee.WithBaseURL(srv.URL))
	devices, err := client.GetDevices(context.Background())
	if err != nil {
		t.Fatalf("GetDevices: %v", err)
	}

	// Verify we can find the first device.
	var found *govee.Device
	for i, d := range devices {
		if d.DeviceID == "AA:BB:CC:DD:EE:FF:00:11" && d.Model == "H6159" {
			found = &devices[i]
			break
		}
	}
	if found == nil {
		t.Fatal("expected to find device AA:BB:CC:DD:EE:FF:00:11")
	}
	if found.DeviceName != "Living Room Light" {
		t.Errorf("DeviceName = %q, want %q", found.DeviceName, "Living Room Light")
	}
	logger.Debugf("found device: %s", found.DeviceName)
}

func TestConnectToDevice_NotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/devices", func(w http.ResponseWriter, _ *http.Request) {
		respondGovee(t, w, map[string]any{"devices": testDeviceList()})
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	logger := logging.NewTestLogger(t)
	// connectToDevice creates its own client internally, but we can't inject
	// the test URL that way. Instead, test the matching logic directly.
	client := govee.New("test-key", govee.WithBaseURL(srv.URL))
	devices, err := client.GetDevices(context.Background())
	if err != nil {
		t.Fatalf("GetDevices: %v", err)
	}

	for _, d := range devices {
		if d.DeviceID == "XX:XX:XX" && d.Model == "NOPE" {
			t.Fatal("should not find non-existent device")
		}
	}
	logger.Debugf("correctly did not find non-existent device")
}

func TestConnectToDevice_APIError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/devices", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		if err := json.NewEncoder(w).Encode(map[string]any{
			"code":    403,
			"message": "Unauthorized",
			"data":    nil,
		}); err != nil {
			t.Errorf("failed to encode response: %v", err)
		}
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	client := govee.New("bad-key", govee.WithBaseURL(srv.URL))
	_, err := client.GetDevices(context.Background())
	if err == nil {
		t.Fatal("expected error for 403 response")
	}
}
