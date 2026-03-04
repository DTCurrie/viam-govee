package light

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	govee "github.com/DTCurrie/govee-go"
	toggleswitch "go.viam.com/rdk/components/switch"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/resource"

	"github.com/DTCurrie/viam-govee/internal/common"
)

// GoveeLightSnapshot is the model identifier for the govee-light-snapshot component.
var GoveeLightSnapshot = common.Family.WithModel("govee-light-snapshot")

func init() {
	resource.RegisterComponent(toggleswitch.API, GoveeLightSnapshot,
		resource.Registration[toggleswitch.Switch, *SnapshotConfig]{
			Constructor: newGoveeLightSnapshot,
		},
	)
}

// SnapshotConfig is the configuration for the govee-light-snapshot component.
type SnapshotConfig struct {
	APIKey   string `json:"api_key"`
	DeviceID string `json:"device"`
	SKU      string `json:"sku"`
}

// Validate checks that all required fields are set.
func (cfg *SnapshotConfig) Validate(_ string) ([]string, []string, error) {
	if cfg.APIKey == "" {
		return nil, nil, fmt.Errorf("api_key is required")
	}
	if cfg.DeviceID == "" {
		return nil, nil, fmt.Errorf("device (MAC address) is required")
	}
	if cfg.SKU == "" {
		return nil, nil, fmt.Errorf("sku is required")
	}
	return nil, nil, nil
}

// snapshotOption is a named snapshot option parsed from device capabilities.
type snapshotOption struct {
	Name  string
	Value int
}

type goveeLightSnapshot struct {
	resource.AlwaysRebuild
	resource.TriviallyCloseable

	name   resource.Name
	logger logging.Logger
	cfg    *SnapshotConfig

	client    *govee.Client
	snapshots []snapshotOption // parsed at construction from static device capabilities
	labels    []string         // position labels: ["off", snapshot1.Name, ...]

	mu       sync.Mutex
	position uint32
}

func newGoveeLightSnapshot(ctx context.Context, _ resource.Dependencies, rawConf resource.Config, logger logging.Logger) (toggleswitch.Switch, error) {
	conf, err := resource.NativeConfig[*SnapshotConfig](rawConf)
	if err != nil {
		return nil, err
	}

	client, device, err := common.ConnectToDevice(ctx, conf.APIKey, conf.DeviceID, conf.SKU, logger)
	if err != nil {
		return nil, err
	}

	snapshots, err := parseSnapshotOptions(device)
	if err != nil {
		return nil, fmt.Errorf("govee: failed to parse snapshot options for %s: %w", conf.DeviceID, err)
	}

	labels := make([]string, 0, len(snapshots)+1)
	labels = append(labels, "off")
	for _, sn := range snapshots {
		labels = append(labels, sn.Name)
	}

	return &goveeLightSnapshot{
		name:      rawConf.ResourceName(),
		logger:    logger,
		cfg:       conf,
		client:    client,
		snapshots: snapshots,
		labels:    labels,
	}, nil
}

// parseSnapshotOptions extracts snapshot options from the device's static capabilities.
// Snapshots are stored as a dynamic_scene/snapshot capability with ENUM parameters.
func parseSnapshotOptions(device *govee.Device) ([]snapshotOption, error) {
	cap := device.FindCapability(govee.CapabilityDynamicScene, "snapshot")
	if cap == nil {
		return nil, fmt.Errorf("snapshot capability not found")
	}
	if cap.Parameters == nil {
		return nil, fmt.Errorf("snapshot capability has no parameters")
	}

	var params govee.EnumParameters
	if err := json.Unmarshal(cap.Parameters, &params); err != nil {
		return nil, fmt.Errorf("failed to parse snapshot parameters: %w", err)
	}

	snapshots := make([]snapshotOption, 0, len(params.Options))
	for _, opt := range params.Options {
		var val int
		if err := json.Unmarshal(opt.Value, &val); err != nil {
			return nil, fmt.Errorf("failed to decode snapshot value for %q: %w", opt.Name, err)
		}
		snapshots = append(snapshots, snapshotOption{Name: opt.Name, Value: val})
	}
	return snapshots, nil
}

func (s *goveeLightSnapshot) Name() resource.Name {
	return s.name
}

func (s *goveeLightSnapshot) DoCommand(_ context.Context, _ map[string]interface{}) (map[string]interface{}, error) {
	return nil, nil
}

// SetPosition activates a snapshot by its position index.
// Position 0 turns the device off. Positions 1..N activate the corresponding
// snapshot parsed from the device's static capabilities at startup.
func (s *goveeLightSnapshot) SetPosition(ctx context.Context, position uint32, _ map[string]interface{}) error {
	if int(position) >= len(s.labels) {
		return fmt.Errorf("invalid position %d, must be 0-%d", position, len(s.labels)-1)
	}

	if position == 0 {
		if err := s.client.TurnOff(ctx, s.cfg.SKU, s.cfg.DeviceID); err != nil {
			return err
		}
		s.mu.Lock()
		s.position = 0
		s.mu.Unlock()
		return nil
	}

	snap := s.snapshots[position-1]

	if err := s.client.TurnOn(ctx, s.cfg.SKU, s.cfg.DeviceID); err != nil {
		return fmt.Errorf("govee: failed to turn on device: %w", err)
	}

	if err := s.client.SetSnapshot(ctx, s.cfg.SKU, s.cfg.DeviceID, snap.Value); err != nil {
		return fmt.Errorf("govee: failed to set snapshot %q: %w", snap.Name, err)
	}

	s.mu.Lock()
	s.position = position
	s.mu.Unlock()
	return nil
}

// GetPosition returns the last position set via SetPosition.
func (s *goveeLightSnapshot) GetPosition(_ context.Context, _ map[string]interface{}) (uint32, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.position, nil
}

// GetNumberOfPositions returns the total number of positions (1 for off + number of snapshots).
func (s *goveeLightSnapshot) GetNumberOfPositions(_ context.Context, _ map[string]interface{}) (uint32, []string, error) {
	return uint32(len(s.labels)), s.labels, nil
}
