package plug

import (
	"context"
	"testing"

	govee "github.com/DTCurrie/govee-go"
	"go.viam.com/rdk/components/sensor"
	"go.viam.com/rdk/logging"

	"github.com/DTCurrie/viam-govee/internal/testutil"
)

func TestPlugSensor_Readings_WithLiveState(t *testing.T) {
	t.Parallel()
	ts := testutil.NewTestServer(t, testutil.TestDeviceList(), testutil.TestStateOn(0, 0, 0, 0))

	s := &goveePlugSensor{
		name:   sensor.Named("test-plug-sensor"),
		logger: logging.NewTestLogger(t),
		cfg: &SensorConfig{
			APIKey:   "test-key",
			DeviceID: "11:22:33:44:55:66:77:88",
			SKU:      "H5081",
		},
		client: ts.Client,
		device: testPlugDevice(),
	}

	readings, err := s.Readings(context.Background(), nil)
	if err != nil {
		t.Fatalf("Readings: %v", err)
	}

	if readings["device_name"] != "Smart Plug" {
		t.Errorf("device_name = %v", readings["device_name"])
	}
	if readings["sku"] != "H5081" {
		t.Errorf("sku = %v", readings["sku"])
	}
	if readings["device_id"] != "11:22:33:44:55:66:77:88" {
		t.Errorf("device_id = %v", readings["device_id"])
	}
	if readings["device_type"] != govee.DeviceSocket {
		t.Errorf("device_type = %v", readings["device_type"])
	}
	if readings["capabilities"] == nil {
		t.Error("capabilities should be present")
	}

	if readings["online"] != true {
		t.Errorf("online = %v", readings["online"])
	}
	if readings["power_state"] != true {
		t.Errorf("power_state = %v, want true", readings["power_state"])
	}

	if _, ok := readings["brightness"]; ok {
		t.Error("plug sensor should not include brightness field")
	}
	if _, ok := readings["red"]; ok {
		t.Error("plug sensor should not include red field")
	}
}

func TestPlugSensor_Readings_NoLiveState(t *testing.T) {
	t.Parallel()
	ts := testutil.NewTestServer(t, testutil.TestDeviceList(), []any{})

	s := &goveePlugSensor{
		name:   sensor.Named("test-plug-sensor"),
		logger: logging.NewTestLogger(t),
		cfg: &SensorConfig{
			APIKey:   "test-key",
			DeviceID: "11:22:33:44:55:66:77:88",
			SKU:      "H5081",
		},
		client: ts.Client,
		device: testPlugDevice(),
	}

	readings, err := s.Readings(context.Background(), nil)
	if err != nil {
		t.Fatalf("Readings: %v", err)
	}

	if readings["device_name"] != "Smart Plug" {
		t.Errorf("device_name = %v", readings["device_name"])
	}
	if readings["online"] != false {
		t.Errorf("online = %v, want false", readings["online"])
	}
	if _, ok := readings["power_state"]; ok {
		t.Error("power_state should not be present when not in state caps")
	}
}

func TestPlugSensor_Readings_StateAPIError(t *testing.T) {
	t.Parallel()
	ts := testutil.NewTestServer(t, testutil.TestDeviceList(), nil)
	ts.StateError = true

	s := &goveePlugSensor{
		name:   sensor.Named("test-plug-sensor"),
		logger: logging.NewTestLogger(t),
		cfg: &SensorConfig{
			APIKey:   "test-key",
			DeviceID: "11:22:33:44:55:66:77:88",
			SKU:      "H5081",
		},
		client: ts.Client,
		device: testPlugDevice(),
	}

	readings, err := s.Readings(context.Background(), nil)
	if err != nil {
		t.Fatalf("Readings should not return error on state failure: %v", err)
	}

	if readings["device_name"] != "Smart Plug" {
		t.Errorf("device_name = %v", readings["device_name"])
	}
	note, ok := readings["note"]
	if !ok {
		t.Fatal("expected 'note' field when state API fails")
	}
	noteStr, ok := note.(string)
	if !ok || noteStr == "" {
		t.Errorf("note should be a non-empty string, got %v", note)
	}
	if _, ok := readings["online"]; ok {
		t.Error("online should not be present when state API fails")
	}
}

func TestPlugSensor_Readings_TogglesPresent(t *testing.T) {
	t.Parallel()
	stateWithToggles := append(testutil.TestStateOn(0, 0, 0, 0),
		map[string]any{
			"type":     "devices.capabilities.toggle",
			"instance": "socketToggle1",
			"state":    map[string]any{"value": 1},
		},
		map[string]any{
			"type":     "devices.capabilities.toggle",
			"instance": "socketToggle2",
			"state":    map[string]any{"value": 0},
		},
	)
	ts := testutil.NewTestServer(t, testutil.TestDeviceList(), stateWithToggles)

	device := testPlugDevice()
	device.Capabilities = append(device.Capabilities,
		govee.Capability{Type: govee.CapabilityToggle, Instance: "socketToggle1"},
		govee.Capability{Type: govee.CapabilityToggle, Instance: "socketToggle2"},
	)

	s := &goveePlugSensor{
		name:   sensor.Named("test-plug-sensor"),
		logger: logging.NewTestLogger(t),
		cfg: &SensorConfig{
			APIKey:   "test-key",
			DeviceID: "11:22:33:44:55:66:77:88",
			SKU:      "H5081",
		},
		client: ts.Client,
		device: device,
	}

	readings, err := s.Readings(context.Background(), nil)
	if err != nil {
		t.Fatalf("Readings: %v", err)
	}

	if readings["toggle_socketToggle1"] != true {
		t.Errorf("toggle_socketToggle1 = %v, want true", readings["toggle_socketToggle1"])
	}
	if readings["toggle_socketToggle2"] != false {
		t.Errorf("toggle_socketToggle2 = %v, want false", readings["toggle_socketToggle2"])
	}
}

func TestSensorConfig_Validate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		cfg     SensorConfig
		wantErr bool
	}{
		{SensorConfig{}, true},
		{SensorConfig{APIKey: "k"}, true},
		{SensorConfig{APIKey: "k", DeviceID: "d"}, true},
		{SensorConfig{APIKey: "k", DeviceID: "d", SKU: "s"}, false},
	}
	for _, tt := range tests {
		_, _, err := tt.cfg.Validate("")
		if (err != nil) != tt.wantErr {
			t.Errorf("Validate(%+v) error = %v, wantErr = %v", tt.cfg, err, tt.wantErr)
		}
	}
}
