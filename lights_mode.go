package viamgovee

import (
	"context"
	"fmt"
	"sync"

	govee "github.com/DTCurrie/govee-go"
	toggleswitch "go.viam.com/rdk/components/switch"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/resource"
)

var GoveeLightMode = family.WithModel("govee-lights-mode")

func init() {
	resource.RegisterComponent(toggleswitch.API, GoveeLightMode,
		resource.Registration[toggleswitch.Switch, *LightModeConfig]{
			Constructor: newGoveeLightMode,
		},
	)
}

// DeviceRef identifies a single Govee device by MAC address and model.
type DeviceRef struct {
	DeviceID string `json:"device"`
	Model    string `json:"model"`
}

// LightModeConfig is the configuration for the govee-lights-mode component.
type LightModeConfig struct {
	APIKey   string      `json:"api_key"`
	Daylight []DeviceRef `json:"daylight,omitempty"`
	Warm     []DeviceRef `json:"warm,omitempty"`
}

func (cfg *LightModeConfig) Validate(path string) ([]string, []string, error) {
	if cfg.APIKey == "" {
		return nil, nil, fmt.Errorf("api_key is required")
	}
	return nil, nil, nil
}

// modeNames maps switch positions to mode names.
var modeNames = []string{"none", "daylight", "warm"}

// savedState holds enough state to restore a device after a mode ends.
type savedState struct {
	PowerState string
	Brightness int
	Color      *govee.Color
	ColorTemp  int
}

type goveeLightMode struct {
	resource.AlwaysRebuild
	resource.TriviallyCloseable

	name   resource.Name
	logger logging.Logger
	cfg    *LightModeConfig

	client *govee.Client

	mu          sync.Mutex
	position    uint32
	savedStates map[string]*savedState // device MAC -> saved state
}

func newGoveeLightMode(ctx context.Context, _ resource.Dependencies, rawConf resource.Config, logger logging.Logger) (toggleswitch.Switch, error) {
	conf, err := resource.NativeConfig[*LightModeConfig](rawConf)
	if err != nil {
		return nil, err
	}

	client := govee.New(conf.APIKey)

	return &goveeLightMode{
		name:        rawConf.ResourceName(),
		logger:      logger,
		cfg:         conf,
		client:      client,
		savedStates: make(map[string]*savedState),
	}, nil
}

func (s *goveeLightMode) Name() resource.Name {
	return s.name
}

func (s *goveeLightMode) DoCommand(ctx context.Context, cmd map[string]interface{}) (map[string]interface{}, error) {
	return nil, nil
}

// SetPosition switches between lighting modes.
// Position 0 = "none" (restore saved state).
// Position 1 = "daylight" (cool white ~6500K, full brightness).
// Position 2 = "warm" (warm white ~2700K, moderate brightness).
func (s *goveeLightMode) SetPosition(ctx context.Context, position uint32, extra map[string]interface{}) error {
	if int(position) >= len(modeNames) {
		return fmt.Errorf("invalid position %d, must be 0-%d", position, len(modeNames)-1)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if position == 0 {
		return s.restoreState(ctx)
	}

	devices := s.devicesForPosition(position)
	if err := s.saveState(ctx, devices); err != nil {
		return err
	}

	switch modeNames[position] {
	case "daylight":
		return s.activateDaylight(ctx, devices, position)
	case "warm":
		return s.activateWarm(ctx, devices, position)
	}

	return fmt.Errorf("unknown mode %q", modeNames[position])
}

func (s *goveeLightMode) GetPosition(ctx context.Context, extra map[string]interface{}) (uint32, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.position, nil
}

func (s *goveeLightMode) GetNumberOfPositions(ctx context.Context, extra map[string]interface{}) (uint32, []string, error) {
	return uint32(len(modeNames)), modeNames, nil
}

// devicesForPosition returns the device list for a given mode position.
func (s *goveeLightMode) devicesForPosition(position uint32) []DeviceRef {
	switch modeNames[position] {
	case "daylight":
		return s.cfg.Daylight
	case "warm":
		return s.cfg.Warm
	}
	return nil
}

// saveState snapshots the current state of each device before activating a mode.
func (s *goveeLightMode) saveState(ctx context.Context, devices []DeviceRef) error {
	s.savedStates = make(map[string]*savedState)
	for _, d := range devices {
		state, err := s.client.GetDeviceState(ctx, d.DeviceID, d.Model)
		if err != nil {
			s.logger.Warnf("govee: failed to save state for device %s: %v", d.DeviceID, err)
			continue
		}
		saved := &savedState{
			PowerState: state.PowerState,
			Brightness: state.Brightness,
			ColorTemp:  state.ColorTemp,
		}
		if state.Color != nil {
			c := *state.Color
			saved.Color = &c
		}
		s.savedStates[d.DeviceID] = saved
	}
	return nil
}

// restoreState returns each device to its pre-mode state.
func (s *goveeLightMode) restoreState(ctx context.Context) error {
	var firstErr error
	for deviceID, saved := range s.savedStates {
		model := s.modelForDevice(deviceID)
		if model == "" {
			continue
		}

		if saved.PowerState == "off" {
			if err := s.client.TurnOff(ctx, deviceID, model); err != nil {
				s.logger.Warnf("govee: failed to turn off %s during restore: %v", deviceID, err)
				if firstErr == nil {
					firstErr = err
				}
			}
			continue
		}

		if err := s.client.TurnOn(ctx, deviceID, model); err != nil {
			s.logger.Warnf("govee: failed to turn on %s during restore: %v", deviceID, err)
			if firstErr == nil {
				firstErr = err
			}
		}

		// Restore color/colorTemp first, then brightness.
		if saved.Color != nil {
			if err := s.client.SetColor(ctx, deviceID, model, saved.Color.R, saved.Color.G, saved.Color.B); err != nil {
				s.logger.Warnf("govee: failed to restore color for %s: %v", deviceID, err)
				if firstErr == nil {
					firstErr = err
				}
			}
		} else if saved.ColorTemp > 0 {
			if err := s.client.SetColorTemp(ctx, deviceID, model, saved.ColorTemp); err != nil {
				s.logger.Warnf("govee: failed to restore color temp for %s: %v", deviceID, err)
				if firstErr == nil {
					firstErr = err
				}
			}
		}

		if saved.Brightness > 0 {
			if err := s.client.SetBrightness(ctx, deviceID, model, saved.Brightness); err != nil {
				s.logger.Warnf("govee: failed to restore brightness for %s: %v", deviceID, err)
				if firstErr == nil {
					firstErr = err
				}
			}
		}
	}
	s.savedStates = make(map[string]*savedState)
	s.position = 0
	return firstErr
}

// modelForDevice looks up the model string for a given device ID across both mode lists.
func (s *goveeLightMode) modelForDevice(deviceID string) string {
	for _, d := range s.cfg.Daylight {
		if d.DeviceID == deviceID {
			return d.Model
		}
	}
	for _, d := range s.cfg.Warm {
		if d.DeviceID == deviceID {
			return d.Model
		}
	}
	return ""
}

// activateDaylight sets each device to a cool daylight white (~6500K) at full brightness.
func (s *goveeLightMode) activateDaylight(ctx context.Context, devices []DeviceRef, position uint32) error {
	for _, d := range devices {
		if err := s.client.TurnOn(ctx, d.DeviceID, d.Model); err != nil {
			return fmt.Errorf("govee: failed to turn on %s: %w", d.DeviceID, err)
		}
		if err := s.client.SetColorTemp(ctx, d.DeviceID, d.Model, 6500); err != nil {
			return fmt.Errorf("govee: failed to set color temp on %s: %w", d.DeviceID, err)
		}
		if err := s.client.SetBrightness(ctx, d.DeviceID, d.Model, 100); err != nil {
			return fmt.Errorf("govee: failed to set brightness on %s: %w", d.DeviceID, err)
		}
	}
	s.position = position
	return nil
}

// activateWarm sets each device to a warm incandescent white (~2700K) at moderate brightness.
func (s *goveeLightMode) activateWarm(ctx context.Context, devices []DeviceRef, position uint32) error {
	for _, d := range devices {
		if err := s.client.TurnOn(ctx, d.DeviceID, d.Model); err != nil {
			return fmt.Errorf("govee: failed to turn on %s: %w", d.DeviceID, err)
		}
		if err := s.client.SetColorTemp(ctx, d.DeviceID, d.Model, 2700); err != nil {
			return fmt.Errorf("govee: failed to set color temp on %s: %w", d.DeviceID, err)
		}
		if err := s.client.SetBrightness(ctx, d.DeviceID, d.Model, 60); err != nil {
			return fmt.Errorf("govee: failed to set brightness on %s: %w", d.DeviceID, err)
		}
	}
	s.position = position
	return nil
}
