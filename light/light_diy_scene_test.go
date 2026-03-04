package light

import (
	"context"
	"testing"

	toggleswitch "go.viam.com/rdk/components/switch"
	"go.viam.com/rdk/logging"

	"github.com/DTCurrie/viam-govee/internal/testutil"
)

const testDIYSceneDeviceID = "AA:BB:CC:DD:EE:FF:00:22"
const testDIYSceneSKU = "H6160"

func newTestLightDIYScene(t *testing.T) (*goveeLightDIYScene, *testutil.TestServer) {
	t.Helper()
	diyScenesByDevice := map[string][]map[string]any{
		testDIYSceneDeviceID: testutil.TestDIYSceneOptions(),
	}
	ts := testutil.NewTestServerFull(t, testutil.TestDeviceList(), nil, nil, diyScenesByDevice)

	goveeScenes, err := ts.Client.GetDIYScenes(context.Background(), testDIYSceneSKU, testDIYSceneDeviceID)
	if err != nil {
		t.Fatalf("GetDIYScenes: %v", err)
	}

	labels := make([]string, 0, len(goveeScenes)+1)
	labels = append(labels, "off")
	for _, sc := range goveeScenes {
		labels = append(labels, sc.Name)
	}

	s := &goveeLightDIYScene{
		name:   toggleswitch.Named("test-diy-scene"),
		logger: logging.NewTestLogger(t),
		cfg: &DIYSceneConfig{
			APIKey:   "test-key",
			DeviceID: testDIYSceneDeviceID,
			SKU:      testDIYSceneSKU,
		},
		client:    ts.Client,
		diyScenes: goveeScenes,
		labels:    labels,
	}
	return s, ts
}

func TestLightDIYScene_GetNumberOfPositions(t *testing.T) {
	t.Parallel()
	s, _ := newTestLightDIYScene(t)

	n, labels, err := s.GetNumberOfPositions(context.Background(), nil)
	if err != nil {
		t.Fatalf("GetNumberOfPositions: %v", err)
	}

	if n != 3 {
		t.Errorf("positions = %d, want 3", n)
	}
	if len(labels) != 3 {
		t.Fatalf("labels length = %d, want 3", len(labels))
	}
	if labels[0] != "off" {
		t.Errorf("labels[0] = %q, want off", labels[0])
	}
	if labels[1] != "My Custom Scene" {
		t.Errorf("labels[1] = %q, want My Custom Scene", labels[1])
	}
	if labels[2] != "Party Mode" {
		t.Errorf("labels[2] = %q, want Party Mode", labels[2])
	}
}

func TestLightDIYScene_SetPosition_Off(t *testing.T) {
	t.Parallel()
	s, ts := newTestLightDIYScene(t)

	if err := s.SetPosition(context.Background(), 0, nil); err != nil {
		t.Fatalf("SetPosition(0): %v", err)
	}

	instances := ts.ControlInstances()
	if len(instances) != 1 || instances[0] != "powerSwitch" {
		t.Errorf("expected [powerSwitch], got %v", instances)
	}

	pos, _ := s.GetPosition(context.Background(), nil)
	if pos != 0 {
		t.Errorf("position = %d, want 0", pos)
	}
}

func TestLightDIYScene_SetPosition_Scene(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		position uint32
		wantPos  uint32
	}{
		{"first_scene", 1, 1},
		{"second_scene", 2, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s, ts := newTestLightDIYScene(t)

			if err := s.SetPosition(context.Background(), tt.position, nil); err != nil {
				t.Fatalf("SetPosition(%d): %v", tt.position, err)
			}

			instances := ts.ControlInstances()
			if len(instances) != 2 {
				t.Fatalf("expected 2 control calls, got %v", instances)
			}
			if instances[0] != "powerSwitch" {
				t.Errorf("call[0] instance = %q, want powerSwitch", instances[0])
			}
			if instances[1] != "diyScene" {
				t.Errorf("call[1] instance = %q, want diyScene", instances[1])
			}

			pos, _ := s.GetPosition(context.Background(), nil)
			if pos != tt.wantPos {
				t.Errorf("position = %d, want %d", pos, tt.wantPos)
			}
		})
	}
}

func TestLightDIYScene_SetPosition_Invalid(t *testing.T) {
	t.Parallel()
	s, _ := newTestLightDIYScene(t)

	if err := s.SetPosition(context.Background(), 99, nil); err == nil {
		t.Error("expected error for invalid position")
	}
}

func TestLightDIYScene_SetPosition_ControlError(t *testing.T) {
	t.Parallel()
	s, ts := newTestLightDIYScene(t)
	ts.ControlFailOn = 2 // TurnOn succeeds, SetDIYScene fails

	if err := s.SetPosition(context.Background(), 1, nil); err == nil {
		t.Error("expected error when SetDIYScene fails")
	}
}

func TestLightDIYScene_GetPosition_Default(t *testing.T) {
	t.Parallel()
	s, _ := newTestLightDIYScene(t)

	pos, err := s.GetPosition(context.Background(), nil)
	if err != nil {
		t.Fatalf("GetPosition: %v", err)
	}
	if pos != 0 {
		t.Errorf("initial position = %d, want 0", pos)
	}
}

func TestDIYSceneConfig_Validate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		cfg     DIYSceneConfig
		wantErr bool
	}{
		{DIYSceneConfig{}, true},
		{DIYSceneConfig{APIKey: "k"}, true},
		{DIYSceneConfig{APIKey: "k", DeviceID: "d"}, true},
		{DIYSceneConfig{APIKey: "k", DeviceID: "d", SKU: "s"}, false},
	}
	for _, tt := range tests {
		_, _, err := tt.cfg.Validate("")
		if (err != nil) != tt.wantErr {
			t.Errorf("Validate(%+v) error = %v, wantErr = %v", tt.cfg, err, tt.wantErr)
		}
	}
}
