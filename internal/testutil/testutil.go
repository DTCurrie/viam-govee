package testutil

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	govee "github.com/DTCurrie/govee-go"
)

// RespondGoveeGet writes a Govee API JSON GET envelope response.
func RespondGoveeGet(t *testing.T, w http.ResponseWriter, data any) {
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

// RespondGoveePost writes a Govee API JSON POST envelope response.
func RespondGoveePost(t *testing.T, w http.ResponseWriter, payload any) {
	t.Helper()
	if payload == nil {
		payload = map[string]any{}
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal response payload: %v", err)
	}
	resp := map[string]any{
		"requestId": "test-request-id",
		"code":      200,
		"msg":       "success",
		"payload":   json.RawMessage(payloadBytes),
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		t.Errorf("failed to encode response: %v", err)
	}
}

// ControlCall records a single POST /device/control request.
type ControlCall struct {
	SKU      string
	DeviceID string
	Type     string
	Instance string
	Value    json.RawMessage
}

// TestServer wraps an httptest.Server that mocks the Govee OpenAPI. It records
// all control calls for later assertions.
type TestServer struct {
	Client *govee.Client

	mu       sync.Mutex
	controls []ControlCall

	scenesByDevice    map[string][]map[string]any
	diyScenesByDevice map[string][]map[string]any

	// StateError when true causes /device/state to return HTTP 500.
	StateError bool
	// ControlFailOn if > 0, the Nth control call (1-indexed) returns HTTP 500
	// instead of succeeding. Earlier calls succeed normally.
	ControlFailOn int
}

// Controls returns a snapshot of all recorded control calls.
func (ts *TestServer) Controls() []ControlCall {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	out := make([]ControlCall, len(ts.controls))
	copy(out, ts.controls)
	return out
}

// ClearControls resets the recorded control calls.
func (ts *TestServer) ClearControls() {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.controls = nil
}

// ControlInstances returns the capability instances of all recorded control calls.
func (ts *TestServer) ControlInstances() []string {
	calls := ts.Controls()
	instances := make([]string, len(calls))
	for i, c := range calls {
		instances[i] = c.Instance
	}
	return instances
}

// NewTestServer creates a mock Govee OpenAPI server.
func NewTestServer(t *testing.T, devices []map[string]any, stateCapabilities []any) *TestServer {
	t.Helper()
	return NewTestServerWithScenes(t, devices, stateCapabilities, nil)
}

// NewTestServerWithScenes creates a mock Govee OpenAPI server with scene support.
func NewTestServerWithScenes(t *testing.T, devices []map[string]any, stateCapabilities []any, scenesByDevice map[string][]map[string]any) *TestServer {
	return NewTestServerFull(t, devices, stateCapabilities, scenesByDevice, nil)
}

// NewTestServerFull creates a mock Govee OpenAPI server with full scene and DIY scene support.
func NewTestServerFull(t *testing.T, devices []map[string]any, stateCapabilities []any, scenesByDevice, diyScenesByDevice map[string][]map[string]any) *TestServer {
	t.Helper()

	ts := &TestServer{
		scenesByDevice:    scenesByDevice,
		diyScenesByDevice: diyScenesByDevice,
	}
	mux := http.NewServeMux()

	mux.HandleFunc("/user/devices", func(w http.ResponseWriter, _ *http.Request) {
		RespondGoveeGet(t, w, devices)
	})

	mux.HandleFunc("/device/state", func(w http.ResponseWriter, r *http.Request) {
		if ts.StateError {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		var outer struct {
			Payload struct {
				SKU      string `json:"sku"`
				DeviceID string `json:"device"`
			} `json:"payload"`
		}
		_ = json.NewDecoder(r.Body).Decode(&outer)

		capabilities := stateCapabilities
		if capabilities == nil {
			capabilities = []any{}
		}
		RespondGoveePost(t, w, map[string]any{
			"sku":          outer.Payload.SKU,
			"device":       outer.Payload.DeviceID,
			"capabilities": capabilities,
		})
	})

	mux.HandleFunc("/device/control", func(w http.ResponseWriter, r *http.Request) {
		var outer struct {
			Payload struct {
				SKU        string `json:"sku"`
				DeviceID   string `json:"device"`
				Capability struct {
					Type     string          `json:"type"`
					Instance string          `json:"instance"`
					Value    json.RawMessage `json:"value"`
				} `json:"capability"`
			} `json:"payload"`
		}
		if err := json.NewDecoder(r.Body).Decode(&outer); err != nil {
			t.Errorf("failed to decode control request: %v", err)
		}

		ts.mu.Lock()
		callNum := len(ts.controls) + 1
		shouldFail := ts.ControlFailOn > 0 && callNum >= ts.ControlFailOn
		ts.mu.Unlock()

		if shouldFail {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		ts.mu.Lock()
		ts.controls = append(ts.controls, ControlCall{
			SKU:      outer.Payload.SKU,
			DeviceID: outer.Payload.DeviceID,
			Type:     outer.Payload.Capability.Type,
			Instance: outer.Payload.Capability.Instance,
			Value:    outer.Payload.Capability.Value,
		})
		ts.mu.Unlock()
		RespondGoveePost(t, w, nil)
	})

	mux.HandleFunc("/device/scenes", func(w http.ResponseWriter, r *http.Request) {
		var outer struct {
			Payload struct {
				SKU      string `json:"sku"`
				DeviceID string `json:"device"`
			} `json:"payload"`
		}
		_ = json.NewDecoder(r.Body).Decode(&outer)
		req := outer.Payload

		var options []map[string]any
		if ts.scenesByDevice != nil {
			options = ts.scenesByDevice[req.DeviceID]
		}
		if options == nil {
			options = []map[string]any{}
		}

		RespondGoveePost(t, w, map[string]any{
			"sku":    req.SKU,
			"device": req.DeviceID,
			"capabilities": []map[string]any{
				{
					"type":     "devices.capabilities.dynamic_scene",
					"instance": "lightScene",
					"parameters": map[string]any{
						"dataType": "ENUM",
						"options":  options,
					},
				},
			},
		})
	})

	mux.HandleFunc("/device/diy-scenes", func(w http.ResponseWriter, r *http.Request) {
		var outer struct {
			Payload struct {
				SKU      string `json:"sku"`
				DeviceID string `json:"device"`
			} `json:"payload"`
		}
		_ = json.NewDecoder(r.Body).Decode(&outer)
		req := outer.Payload

		var options []map[string]any
		if ts.diyScenesByDevice != nil {
			options = ts.diyScenesByDevice[req.DeviceID]
		}
		if options == nil {
			options = []map[string]any{}
		}

		RespondGoveePost(t, w, map[string]any{
			"sku":    req.SKU,
			"device": req.DeviceID,
			"capabilities": []map[string]any{
				{
					"type":     "devices.capabilities.dynamic_scene",
					"instance": "diyScene",
					"parameters": map[string]any{
						"dataType": "ENUM",
						"options":  options,
					},
				},
			},
		})
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	ts.Client = govee.New("test-key", govee.WithBaseURL(srv.URL))
	return ts
}

// TestDeviceList returns a standard two-device list for use in tests.
// Device 0: a light (H6159) with color, brightness, color temp, scene, snapshot, and toggle capabilities.
// Device 1: a smart plug (H5081) with on/off only.
func TestDeviceList() []map[string]any {
	return []map[string]any{
		{
			"sku":        "H6159",
			"device":     "AA:BB:CC:DD:EE:FF:00:11",
			"deviceName": "Living Room Light",
			"type":       "devices.types.light",
			"capabilities": []map[string]any{
				{"type": "devices.capabilities.on_off", "instance": "powerSwitch"},
				{"type": "devices.capabilities.range", "instance": "brightness"},
				{"type": "devices.capabilities.color_setting", "instance": "colorRgb"},
				{
					"type":     "devices.capabilities.color_setting",
					"instance": "colorTemperatureK",
					"parameters": map[string]any{
						"dataType": "INTEGER",
						"range":    map[string]any{"min": 2000, "max": 9000, "precision": 1},
					},
				},
				{"type": "devices.capabilities.dynamic_scene", "instance": "lightScene"},
				{"type": "devices.capabilities.dynamic_scene", "instance": "diyScene"},
				{
					"type":     "devices.capabilities.dynamic_scene",
					"instance": "snapshot",
					"parameters": map[string]any{
						"dataType": "ENUM",
						"options":  TestSnapshotOptions(),
					},
				},
				{"type": "devices.capabilities.toggle", "instance": "gradientToggle"},
			},
		},
		{
			"sku":        "H5081",
			"device":     "11:22:33:44:55:66:77:88",
			"deviceName": "Smart Plug",
			"type":       "devices.types.socket",
			"capabilities": []map[string]any{
				{"type": "devices.capabilities.on_off", "instance": "powerSwitch"},
			},
		},
	}
}

// TestStateOn returns capability states for a device that is on with color set.
func TestStateOn(brightness int, r, g, b int) []any {
	packed := (r << 16) | (g << 8) | b
	return []any{
		map[string]any{
			"type":     "devices.capabilities.online",
			"instance": "online",
			"state":    map[string]any{"value": true},
		},
		map[string]any{
			"type":     "devices.capabilities.on_off",
			"instance": "powerSwitch",
			"state":    map[string]any{"value": 1},
		},
		map[string]any{
			"type":     "devices.capabilities.range",
			"instance": "brightness",
			"state":    map[string]any{"value": brightness},
		},
		map[string]any{
			"type":     "devices.capabilities.color_setting",
			"instance": "colorRgb",
			"state":    map[string]any{"value": packed},
		},
		map[string]any{
			"type":     "devices.capabilities.color_setting",
			"instance": "colorTemperatureK",
			"state":    map[string]any{"value": 5000},
		},
	}
}

// TestStateOff returns capability states for a device that is off.
func TestStateOff() []any {
	return []any{
		map[string]any{
			"type":     "devices.capabilities.online",
			"instance": "online",
			"state":    map[string]any{"value": true},
		},
		map[string]any{
			"type":     "devices.capabilities.on_off",
			"instance": "powerSwitch",
			"state":    map[string]any{"value": 0},
		},
		map[string]any{
			"type":     "devices.capabilities.range",
			"instance": "brightness",
			"state":    map[string]any{"value": 0},
		},
	}
}

// TestSceneOptions returns a set of mock dynamic scene options.
func TestSceneOptions() []map[string]any {
	return []map[string]any{
		{"name": "Sunrise", "value": map[string]any{"id": 1, "paramId": 10}},
		{"name": "Ocean", "value": map[string]any{"id": 2, "paramId": 20}},
		{"name": "Forest", "value": map[string]any{"id": 3, "paramId": 30}},
	}
}

// TestDIYSceneOptions returns a set of mock DIY scene options.
func TestDIYSceneOptions() []map[string]any {
	return []map[string]any{
		{"name": "My Custom Scene", "value": json.RawMessage(`101`)},
		{"name": "Party Mode", "value": json.RawMessage(`102`)},
	}
}

// TestSnapshotOptions returns a set of mock snapshot options.
func TestSnapshotOptions() []map[string]any {
	return []map[string]any{
		{"name": "Sunrise", "value": json.RawMessage(`0`)},
		{"name": "Sunset", "value": json.RawMessage(`1`)},
	}
}

// ParseJSON is a test helper that unmarshals a json.RawMessage into v.
func ParseJSON(raw json.RawMessage, v any) error {
	return json.Unmarshal(raw, v)
}
