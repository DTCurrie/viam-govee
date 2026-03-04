package light

import (
	"context"
	"testing"

	govee "github.com/DTCurrie/govee-go"
	toggleswitch "go.viam.com/rdk/components/switch"
	"go.viam.com/rdk/logging"

	"github.com/DTCurrie/viam-govee/internal/testutil"
)

const testSnapshotDeviceID = "AA:BB:CC:DD:EE:FF:00:11"
const testSnapshotSKU = "H6159"

func newTestLightSnapshot(t *testing.T) (*goveeLightSnapshot, *testutil.TestServer) {
	t.Helper()
	ts := testutil.NewTestServer(t, testutil.TestDeviceList(), nil)

	snapshots := []snapshotOption{
		{Name: "Sunrise", Value: 0},
		{Name: "Sunset", Value: 1},
	}
	labels := make([]string, 0, len(snapshots)+1)
	labels = append(labels, "off")
	for _, sn := range snapshots {
		labels = append(labels, sn.Name)
	}

	s := &goveeLightSnapshot{
		name:   toggleswitch.Named("test-snapshot"),
		logger: logging.NewTestLogger(t),
		cfg: &SnapshotConfig{
			APIKey:   "test-key",
			DeviceID: testSnapshotDeviceID,
			SKU:      testSnapshotSKU,
		},
		client:    ts.Client,
		snapshots: snapshots,
		labels:    labels,
	}
	return s, ts
}

func TestLightSnapshot_GetNumberOfPositions(t *testing.T) {
	t.Parallel()
	s, _ := newTestLightSnapshot(t)

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
	if labels[1] != "Sunrise" {
		t.Errorf("labels[1] = %q, want Sunrise", labels[1])
	}
	if labels[2] != "Sunset" {
		t.Errorf("labels[2] = %q, want Sunset", labels[2])
	}
}

func TestLightSnapshot_SetPosition_Off(t *testing.T) {
	t.Parallel()
	s, ts := newTestLightSnapshot(t)

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

func TestLightSnapshot_SetPosition_Snapshot(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		position  uint32
		wantValue int
	}{
		{"Sunrise", 1, 0},
		{"Sunset", 2, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s, ts := newTestLightSnapshot(t)

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
			if instances[1] != "snapshot" {
				t.Errorf("call[1] instance = %q, want snapshot", instances[1])
			}

			calls := ts.Controls()
			var val int
			if err := testutil.ParseJSON(calls[1].Value, &val); err != nil {
				t.Fatalf("failed to parse snapshot value: %v", err)
			}
			if val != tt.wantValue {
				t.Errorf("snapshot value = %d, want %d (%s)", val, tt.wantValue, tt.name)
			}

			pos, _ := s.GetPosition(context.Background(), nil)
			if pos != tt.position {
				t.Errorf("position = %d, want %d", pos, tt.position)
			}
		})
	}
}

func TestLightSnapshot_SetPosition_Invalid(t *testing.T) {
	t.Parallel()
	s, _ := newTestLightSnapshot(t)

	if err := s.SetPosition(context.Background(), 99, nil); err == nil {
		t.Error("expected error for invalid position")
	}
}

func TestLightSnapshot_SetPosition_ControlError(t *testing.T) {
	t.Parallel()
	s, ts := newTestLightSnapshot(t)
	ts.ControlFailOn = 2 // TurnOn succeeds, SetSnapshot fails

	if err := s.SetPosition(context.Background(), 1, nil); err == nil {
		t.Error("expected error when SetSnapshot fails")
	}
}

func TestLightSnapshot_GetPosition_Default(t *testing.T) {
	t.Parallel()
	s, _ := newTestLightSnapshot(t)

	pos, err := s.GetPosition(context.Background(), nil)
	if err != nil {
		t.Fatalf("GetPosition: %v", err)
	}
	if pos != 0 {
		t.Errorf("initial position = %d, want 0", pos)
	}
}

func TestSnapshotConfig_Validate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		cfg     SnapshotConfig
		wantErr bool
	}{
		{SnapshotConfig{}, true},
		{SnapshotConfig{APIKey: "k"}, true},
		{SnapshotConfig{APIKey: "k", DeviceID: "d"}, true},
		{SnapshotConfig{APIKey: "k", DeviceID: "d", SKU: "s"}, false},
	}
	for _, tt := range tests {
		_, _, err := tt.cfg.Validate("")
		if (err != nil) != tt.wantErr {
			t.Errorf("Validate(%+v) error = %v, wantErr = %v", tt.cfg, err, tt.wantErr)
		}
	}
}

func TestParseSnapshotOptions(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		device    *govee.Device
		wantNames []string
		wantVals  []int
		wantErr   bool
	}{
		{
			name: "valid options",
			device: &govee.Device{Capabilities: []govee.Capability{
				{
					Type:       govee.CapabilityDynamicScene,
					Instance:   "snapshot",
					Parameters: []byte(`{"dataType":"ENUM","options":[{"name":"Sunrise","value":0},{"name":"Sunset","value":1}]}`),
				},
			}},
			wantNames: []string{"Sunrise", "Sunset"},
			wantVals:  []int{0, 1},
		},
		{
			name:    "no snapshot capability",
			device:  &govee.Device{},
			wantErr: true,
		},
		{
			name: "no parameters",
			device: &govee.Device{Capabilities: []govee.Capability{
				{
					Type:     govee.CapabilityDynamicScene,
					Instance: "snapshot",
				},
			}},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			snaps, err := parseSnapshotOptions(tt.device)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseSnapshotOptions() error = %v, wantErr = %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}
			if len(snaps) != len(tt.wantNames) {
				t.Fatalf("got %d snapshots, want %d", len(snaps), len(tt.wantNames))
			}
			for i, sn := range snaps {
				if sn.Name != tt.wantNames[i] {
					t.Errorf("snapshot[%d].Name = %q, want %q", i, sn.Name, tt.wantNames[i])
				}
				if sn.Value != tt.wantVals[i] {
					t.Errorf("snapshot[%d].Value = %d, want %d", i, sn.Value, tt.wantVals[i])
				}
			}
		})
	}
}
