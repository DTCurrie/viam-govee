package viamgovee

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	govee "github.com/DTCurrie/govee-go"
)

// respondGovee writes a Govee API JSON envelope response.
func respondGovee(t *testing.T, w http.ResponseWriter, data any) {
	t.Helper()
	if data == nil {
		data = map[string]any{}
	}
	dataBytes, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("failed to marshal response data: %v", err)
	}
	resp := map[string]any{
		"code":    200,
		"message": "Success",
		"data":    json.RawMessage(dataBytes),
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		t.Errorf("failed to encode response: %v", err)
	}
}

// controlCall records a single PUT /v1/devices/control request.
type controlCall struct {
	Device string
	Model  string
	Name   string
	Value  json.RawMessage
}

// testServer wraps an httptest.Server that mocks the Govee API. It records
// all control calls for later assertions.
type testServer struct {
	Client *govee.Client

	mu       sync.Mutex
	controls []controlCall
}

// Controls returns a snapshot of all recorded control calls.
func (ts *testServer) Controls() []controlCall {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	out := make([]controlCall, len(ts.controls))
	copy(out, ts.controls)
	return out
}

// ClearControls resets the recorded control calls.
func (ts *testServer) ClearControls() {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.controls = nil
}

// ControlNames returns just the command names of all recorded calls.
func (ts *testServer) ControlNames() []string {
	calls := ts.Controls()
	names := make([]string, len(calls))
	for i, c := range calls {
		names[i] = c.Name
	}
	return names
}

// newTestServer creates a mock Govee API server.
//   - devices is returned by GET /v1/devices.
//   - stateProps is returned by GET /v1/devices/state (the properties array).
func newTestServer(t *testing.T, devices []map[string]any, stateProps []any) *testServer {
	t.Helper()

	ts := &testServer{}
	mux := http.NewServeMux()

	mux.HandleFunc("/v1/devices", func(w http.ResponseWriter, r *http.Request) {
		respondGovee(t, w, map[string]any{"devices": devices})
	})

	mux.HandleFunc("/v1/devices/state", func(w http.ResponseWriter, r *http.Request) {
		respondGovee(t, w, map[string]any{
			"device":     r.URL.Query().Get("device"),
			"model":      r.URL.Query().Get("model"),
			"properties": stateProps,
		})
	})

	mux.HandleFunc("/v1/devices/control", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Device string `json:"device"`
			Model  string `json:"model"`
			Cmd    struct {
				Name  string          `json:"name"`
				Value json.RawMessage `json:"value"`
			} `json:"cmd"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("failed to decode control request: %v", err)
		}
		ts.mu.Lock()
		ts.controls = append(ts.controls, controlCall{
			Device: req.Device,
			Model:  req.Model,
			Name:   req.Cmd.Name,
			Value:  req.Cmd.Value,
		})
		ts.mu.Unlock()
		respondGovee(t, w, nil)
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	ts.Client = govee.New("test-key", govee.WithBaseURL(srv.URL))
	return ts
}

// testDeviceList returns a standard two-device list for use in tests.
func testDeviceList() []map[string]any {
	return []map[string]any{
		{
			"device":       "AA:BB:CC:DD:EE:FF:00:11",
			"model":        "H6159",
			"deviceName":   "Living Room Light",
			"controllable": true,
			"retrievable":  true,
			"supportCmds":  []string{"turn", "brightness", "color", "colorTem"},
			"properties": map[string]any{
				"colorTem": map[string]any{
					"range": map[string]any{"min": 2000, "max": 9000},
				},
			},
		},
		{
			"device":       "11:22:33:44:55:66:77:88",
			"model":        "H5081",
			"deviceName":   "Smart Plug",
			"controllable": true,
			"retrievable":  false,
			"supportCmds":  []string{"turn"},
			"properties":   map[string]any{},
		},
	}
}

// testStateOn returns state properties for a device that is on with color set.
func testStateOn(brightness int, r, g, b int) []any {
	return []any{
		map[string]any{"online": "true"},
		map[string]any{"powerState": "on"},
		map[string]any{"brightness": brightness},
		map[string]any{"color": map[string]any{"r": r, "g": g, "b": b}},
		map[string]any{"colorTem": 5000},
	}
}

// parseJSON is a test helper that unmarshals a json.RawMessage into v.
func parseJSON(raw json.RawMessage, v any) error {
	return json.Unmarshal(raw, v)
}

// testStateOff returns state properties for a device that is off.
func testStateOff() []any {
	return []any{
		map[string]any{"online": "true"},
		map[string]any{"powerState": "off"},
		map[string]any{"brightness": 0},
	}
}
