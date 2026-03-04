package light

import (
	"context"
	"testing"

	govee "github.com/DTCurrie/govee-go"
	"go.viam.com/rdk/components/sensor"
	"go.viam.com/rdk/logging"

	"github.com/DTCurrie/viam-govee/internal/testutil"
)

func testLightDevice() *govee.Device {
	return &govee.Device{
		DeviceID:   "AA:BB:CC:DD:EE:FF:00:11",
		SKU:        "H6159",
		DeviceName: "Living Room Light",
		Type:       govee.DeviceLight,
		Capabilities: []govee.Capability{
			{Type: govee.CapabilityOnOff, Instance: "powerSwitch"},
			{Type: govee.CapabilityRange, Instance: "brightness"},
			{Type: govee.CapabilityColorSetting, Instance: "colorRgb"},
			{Type: govee.CapabilityColorSetting, Instance: "colorTemperatureK"},
			{Type: govee.CapabilityDynamicScene, Instance: "lightScene"},
		},
	}
}

func TestSensor_Readings_WithLiveState(t *testing.T) {
	t.Parallel()
	ts := testutil.NewTestServer(t, testutil.TestDeviceList(), testutil.TestStateOn(80, 255, 128, 0))

	s := &goveeLightSensor{
		name:   sensor.Named("test-sensor"),
		logger: logging.NewTestLogger(t),
		cfg: &SensorConfig{
			APIKey:   "test-key",
			DeviceID: "AA:BB:CC:DD:EE:FF:00:11",
			SKU:      "H6159",
		},
		client: ts.Client,
		device: testLightDevice(),
	}

	readings, err := s.Readings(context.Background(), nil)
	if err != nil {
		t.Fatalf("Readings: %v", err)
	}

	if readings["device_name"] != "Living Room Light" {
		t.Errorf("device_name = %v", readings["device_name"])
	}
	if readings["sku"] != "H6159" {
		t.Errorf("sku = %v", readings["sku"])
	}
	if readings["device_type"] != govee.DeviceLight {
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
	if readings["brightness"] != 80 {
		t.Errorf("brightness = %v", readings["brightness"])
	}
	if readings["red"] != 255 {
		t.Errorf("red = %v", readings["red"])
	}
	if readings["green"] != 128 {
		t.Errorf("green = %v", readings["green"])
	}
	if readings["blue"] != 0 {
		t.Errorf("blue = %v", readings["blue"])
	}
	if readings["color_temp"] != 5000 {
		t.Errorf("color_temp = %v, want 5000", readings["color_temp"])
	}
}

func TestSensor_Readings_NoLiveState(t *testing.T) {
	t.Parallel()
	ts := testutil.NewTestServer(t, testutil.TestDeviceList(), []any{})

	s := &goveeLightSensor{
		name:   sensor.Named("test-sensor"),
		logger: logging.NewTestLogger(t),
		cfg: &SensorConfig{
			APIKey:   "test-key",
			DeviceID: "AA:BB:CC:DD:EE:FF:00:11",
			SKU:      "H6159",
		},
		client: ts.Client,
		device: testLightDevice(),
	}

	readings, err := s.Readings(context.Background(), nil)
	if err != nil {
		t.Fatalf("Readings: %v", err)
	}

	if readings["device_name"] != "Living Room Light" {
		t.Errorf("device_name = %v", readings["device_name"])
	}
	if readings["online"] != false {
		t.Errorf("online = %v, want false", readings["online"])
	}
	if _, ok := readings["power_state"]; ok {
		t.Error("power_state should not be present when not in state caps")
	}
}

func TestSensor_Readings_StateAPIError(t *testing.T) {
	t.Parallel()
	ts := testutil.NewTestServer(t, testutil.TestDeviceList(), nil)
	ts.StateError = true

	s := &goveeLightSensor{
		name:   sensor.Named("test-sensor"),
		logger: logging.NewTestLogger(t),
		cfg: &SensorConfig{
			APIKey:   "test-key",
			DeviceID: "AA:BB:CC:DD:EE:FF:00:11",
			SKU:      "H6159",
		},
		client: ts.Client,
		device: testLightDevice(),
	}

	readings, err := s.Readings(context.Background(), nil)
	if err != nil {
		t.Fatalf("Readings should not return error on state failure: %v", err)
	}

	if readings["device_name"] != "Living Room Light" {
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

func TestSensor_Readings_TogglesPresent(t *testing.T) {
	t.Parallel()
	stateWithToggles := append(testutil.TestStateOn(80, 255, 128, 0),
		map[string]any{
			"type":     "devices.capabilities.toggle",
			"instance": "gradientToggle",
			"state":    map[string]any{"value": 1},
		},
		map[string]any{
			"type":     "devices.capabilities.toggle",
			"instance": "dreamViewToggle",
			"state":    map[string]any{"value": 0},
		},
	)
	ts := testutil.NewTestServer(t, testutil.TestDeviceList(), stateWithToggles)

	device := testLightDevice()
	device.Capabilities = append(device.Capabilities,
		govee.Capability{Type: govee.CapabilityToggle, Instance: "gradientToggle"},
		govee.Capability{Type: govee.CapabilityToggle, Instance: "dreamViewToggle"},
	)

	s := &goveeLightSensor{
		name:   sensor.Named("test-sensor"),
		logger: logging.NewTestLogger(t),
		cfg: &SensorConfig{
			APIKey:   "test-key",
			DeviceID: "AA:BB:CC:DD:EE:FF:00:11",
			SKU:      "H6159",
		},
		client: ts.Client,
		device: device,
	}

	readings, err := s.Readings(context.Background(), nil)
	if err != nil {
		t.Fatalf("Readings: %v", err)
	}

	if readings["toggle_gradientToggle"] != true {
		t.Errorf("toggle_gradientToggle = %v, want true", readings["toggle_gradientToggle"])
	}
	if readings["toggle_dreamViewToggle"] != false {
		t.Errorf("toggle_dreamViewToggle = %v, want false", readings["toggle_dreamViewToggle"])
	}
}

func TestSensor_Readings_NoTogglesInState(t *testing.T) {
	t.Parallel()
	ts := testutil.NewTestServer(t, testutil.TestDeviceList(), testutil.TestStateOn(80, 255, 0, 0))

	device := testLightDevice()
	device.Capabilities = append(device.Capabilities,
		govee.Capability{Type: govee.CapabilityToggle, Instance: "gradientToggle"},
	)

	s := &goveeLightSensor{
		name:   sensor.Named("test-sensor"),
		logger: logging.NewTestLogger(t),
		cfg: &SensorConfig{
			APIKey:   "test-key",
			DeviceID: "AA:BB:CC:DD:EE:FF:00:11",
			SKU:      "H6159",
		},
		client: ts.Client,
		device: device,
	}

	readings, err := s.Readings(context.Background(), nil)
	if err != nil {
		t.Fatalf("Readings: %v", err)
	}

	if _, ok := readings["toggle_gradientToggle"]; ok {
		t.Error("toggle_gradientToggle should not be present when not in state caps")
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
