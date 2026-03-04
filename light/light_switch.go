package light

import (
	"context"
	"fmt"
	"sync"

	govee "github.com/DTCurrie/govee-go"
	toggleswitch "go.viam.com/rdk/components/switch"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/resource"

	"github.com/DTCurrie/viam-govee/internal/common"
)

// GoveeLightSwitch is the model identifier for the govee-light-switch component.
var GoveeLightSwitch = common.Family.WithModel("govee-light-switch")

func init() {
	resource.RegisterComponent(toggleswitch.API, GoveeLightSwitch,
		resource.Registration[toggleswitch.Switch, *SwitchConfig]{
			Constructor: newGoveeLightSwitch,
		},
	)
}

// SwitchConfig is the configuration for the govee-light-switch component.
type SwitchConfig struct {
	APIKey   string `json:"api_key"`
	DeviceID string `json:"device"`
	SKU      string `json:"sku"`
}

// Validate checks that all required fields are set.
func (cfg *SwitchConfig) Validate(_ string) ([]string, []string, error) {
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

type goveeLightSwitch struct {
	resource.AlwaysRebuild
	resource.TriviallyCloseable

	name   resource.Name
	logger logging.Logger
	cfg    *SwitchConfig

	client  *govee.Client
	mu      sync.Mutex
	lastPos uint32 // tracks last known position, used as fallback if state query fails
}

func newGoveeLightSwitch(ctx context.Context, _ resource.Dependencies, rawConf resource.Config, logger logging.Logger) (toggleswitch.Switch, error) {
	conf, err := resource.NativeConfig[*SwitchConfig](rawConf)
	if err != nil {
		return nil, err
	}

	client, _, err := common.ConnectToDevice(ctx, conf.APIKey, conf.DeviceID, conf.SKU, logger)
	if err != nil {
		return nil, err
	}

	s := &goveeLightSwitch{
		name:   rawConf.ResourceName(),
		logger: logger,
		cfg:    conf,
		client: client,
	}

	// Seed lastPos from current device state if available.
	if state, err := client.GetDeviceState(ctx, conf.SKU, conf.DeviceID); err == nil {
		if on, ok := state.PowerState(); ok && on {
			s.lastPos = 1
		}
	}

	return s, nil
}

func (s *goveeLightSwitch) Name() resource.Name {
	return s.name
}

func (s *goveeLightSwitch) DoCommand(_ context.Context, _ map[string]interface{}) (map[string]interface{}, error) {
	return nil, nil
}

// SetPosition controls the light's power state.
// 0 = off. 1 = on.
func (s *goveeLightSwitch) SetPosition(ctx context.Context, position uint32, _ map[string]interface{}) error {
	if position > 1 {
		return fmt.Errorf("position must be 0 (off) or 1 (on), got %d", position)
	}

	var err error
	if position == 0 {
		err = s.client.TurnOff(ctx, s.cfg.SKU, s.cfg.DeviceID)
	} else {
		err = s.client.TurnOn(ctx, s.cfg.SKU, s.cfg.DeviceID)
	}
	if err != nil {
		return err
	}

	s.mu.Lock()
	s.lastPos = position
	s.mu.Unlock()
	return nil
}

// GetPosition returns the current power state of the light as 0 (off) or 1 (on).
// Attempts to query live state from the API; falls back to the last locally tracked
// position if the state query fails.
func (s *goveeLightSwitch) GetPosition(ctx context.Context, _ map[string]interface{}) (uint32, error) {
	if state, err := s.client.GetDeviceState(ctx, s.cfg.SKU, s.cfg.DeviceID); err == nil {
		if on, ok := state.PowerState(); ok {
			if on {
				return 1, nil
			}
			return 0, nil
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lastPos, nil
}

func (s *goveeLightSwitch) GetNumberOfPositions(_ context.Context, _ map[string]interface{}) (uint32, []string, error) {
	return 2, []string{"off", "on"}, nil
}
