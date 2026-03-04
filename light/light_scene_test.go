package light

import (
	"context"
	"testing"

	toggleswitch "go.viam.com/rdk/components/switch"
	"go.viam.com/rdk/logging"

	"github.com/DTCurrie/viam-govee/internal/testutil"
)

const testSceneDeviceID = "AA:BB:CC:DD:EE:FF:00:11"
const testSceneSKU = "H6159"

func newTestLightScene(t *testing.T) (*goveeLightScene, *testutil.TestServer) {
	t.Helper()
	scenesByDevice := map[string][]map[string]any{
		testSceneDeviceID: testutil.TestSceneOptions(),
	}
	ts := testutil.NewTestServerWithScenes(t, testutil.TestDeviceList(), nil, scenesByDevice)

	goveeScenes, err := ts.Client.GetScenes(context.Background(), testSceneSKU, testSceneDeviceID)
	if err != nil {
		t.Fatalf("GetScenes: %v", err)
	}

	labels := make([]string, 0, len(goveeScenes)+1)
	labels = append(labels, "off")
	for _, sc := range goveeScenes {
		labels = append(labels, sc.Name)
	}

	s := &goveeLightScene{
		name:   toggleswitch.Named("test-scene"),
		logger: logging.NewTestLogger(t),
		cfg: &SceneConfig{
			APIKey:   "test-key",
			DeviceID: testSceneDeviceID,
			SKU:      testSceneSKU,
		},
		client: ts.Client,
		scenes: goveeScenes,
		labels: labels,
	}
	return s, ts
}

func TestLightScene_GetNumberOfPositions(t *testing.T) {
	t.Parallel()
	s, _ := newTestLightScene(t)

	n, labels, err := s.GetNumberOfPositions(context.Background(), nil)
	if err != nil {
		t.Fatalf("GetNumberOfPositions: %v", err)
	}

	if n != 4 {
		t.Errorf("positions = %d, want 4", n)
	}
	if len(labels) != 4 {
		t.Fatalf("labels length = %d, want 4", len(labels))
	}
	if labels[0] != "off" {
		t.Errorf("labels[0] = %q, want off", labels[0])
	}
	if labels[1] != "Sunrise" {
		t.Errorf("labels[1] = %q, want Sunrise", labels[1])
	}
	if labels[2] != "Ocean" {
		t.Errorf("labels[2] = %q, want Ocean", labels[2])
	}
	if labels[3] != "Forest" {
		t.Errorf("labels[3] = %q, want Forest", labels[3])
	}
}

func TestLightScene_SetPosition_Off(t *testing.T) {
	t.Parallel()
	s, ts := newTestLightScene(t)

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

func TestLightScene_SetPosition_Scene(t *testing.T) {
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
			s, ts := newTestLightScene(t)

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
			if instances[1] != "lightScene" {
				t.Errorf("call[1] instance = %q, want lightScene", instances[1])
			}

			pos, _ := s.GetPosition(context.Background(), nil)
			if pos != tt.wantPos {
				t.Errorf("position = %d, want %d", pos, tt.wantPos)
			}
		})
	}
}

func TestLightScene_SetPosition_Invalid(t *testing.T) {
	t.Parallel()
	s, _ := newTestLightScene(t)

	if err := s.SetPosition(context.Background(), 99, nil); err == nil {
		t.Error("expected error for invalid position")
	}
}

func TestLightScene_SetPosition_ControlError(t *testing.T) {
	t.Parallel()
	s, ts := newTestLightScene(t)
	ts.ControlFailOn = 2 // TurnOn succeeds, SetLightScene fails

	if err := s.SetPosition(context.Background(), 1, nil); err == nil {
		t.Error("expected error when SetLightScene fails")
	}
}

func TestLightScene_GetPosition_Default(t *testing.T) {
	t.Parallel()
	s, _ := newTestLightScene(t)

	pos, err := s.GetPosition(context.Background(), nil)
	if err != nil {
		t.Fatalf("GetPosition: %v", err)
	}
	if pos != 0 {
		t.Errorf("initial position = %d, want 0", pos)
	}
}

func TestSceneConfig_Validate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		cfg     SceneConfig
		wantErr bool
	}{
		{SceneConfig{}, true},
		{SceneConfig{APIKey: "k"}, true},
		{SceneConfig{APIKey: "k", DeviceID: "d"}, true},
		{SceneConfig{APIKey: "k", DeviceID: "d", SKU: "s"}, false},
	}
	for _, tt := range tests {
		_, _, err := tt.cfg.Validate("")
		if (err != nil) != tt.wantErr {
			t.Errorf("Validate(%+v) error = %v, wantErr = %v", tt.cfg, err, tt.wantErr)
		}
	}
}
