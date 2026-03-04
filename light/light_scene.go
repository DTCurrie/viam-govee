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

// GoveeLightScene is the model identifier for the govee-light-scene component.
var GoveeLightScene = common.Family.WithModel("govee-light-scene")

func init() {
	resource.RegisterComponent(toggleswitch.API, GoveeLightScene,
		resource.Registration[toggleswitch.Switch, *SceneConfig]{
			Constructor: newGoveeLightScene,
		},
	)
}

// SceneConfig is the configuration for the govee-light-scene component.
type SceneConfig struct {
	APIKey   string `json:"api_key"`
	DeviceID string `json:"device"`
	SKU      string `json:"sku"`
}

// Validate checks that all required fields are set.
func (cfg *SceneConfig) Validate(_ string) ([]string, []string, error) {
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

type goveeLightScene struct {
	resource.AlwaysRebuild
	resource.TriviallyCloseable

	name   resource.Name
	logger logging.Logger
	cfg    *SceneConfig

	client *govee.Client
	scenes []govee.SceneOption // cached at construction time
	labels []string            // position labels: ["off", scene1.Name, ...]

	mu       sync.Mutex
	position uint32
}

func newGoveeLightScene(ctx context.Context, _ resource.Dependencies, rawConf resource.Config, logger logging.Logger) (toggleswitch.Switch, error) {
	conf, err := resource.NativeConfig[*SceneConfig](rawConf)
	if err != nil {
		return nil, err
	}

	client, _, err := common.ConnectToDevice(ctx, conf.APIKey, conf.DeviceID, conf.SKU, logger)
	if err != nil {
		return nil, err
	}

	scenes, err := client.GetScenes(ctx, conf.SKU, conf.DeviceID)
	if err != nil {
		return nil, fmt.Errorf("govee: failed to fetch scenes for %s: %w", conf.DeviceID, err)
	}

	labels := make([]string, 0, len(scenes)+1)
	labels = append(labels, "off")
	for _, sc := range scenes {
		labels = append(labels, sc.Name)
	}

	return &goveeLightScene{
		name:   rawConf.ResourceName(),
		logger: logger,
		cfg:    conf,
		client: client,
		scenes: scenes,
		labels: labels,
	}, nil
}

func (s *goveeLightScene) Name() resource.Name {
	return s.name
}

func (s *goveeLightScene) DoCommand(_ context.Context, _ map[string]interface{}) (map[string]interface{}, error) {
	return nil, nil
}

// SetPosition activates a scene by its position index.
// Position 0 turns the device off. Positions 1..N activate the corresponding
// dynamic scene returned by the Govee API (in the order reported at startup).
func (s *goveeLightScene) SetPosition(ctx context.Context, position uint32, _ map[string]interface{}) error {
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

	scene := s.scenes[position-1]

	if err := s.client.TurnOn(ctx, s.cfg.SKU, s.cfg.DeviceID); err != nil {
		return fmt.Errorf("govee: failed to turn on device: %w", err)
	}

	if err := s.client.SetLightScene(ctx, s.cfg.SKU, s.cfg.DeviceID, scene.Value); err != nil {
		return fmt.Errorf("govee: failed to set scene %q: %w", scene.Name, err)
	}

	s.mu.Lock()
	s.position = position
	s.mu.Unlock()
	return nil
}

// GetPosition returns the last position set via SetPosition.
func (s *goveeLightScene) GetPosition(_ context.Context, _ map[string]interface{}) (uint32, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.position, nil
}

// GetNumberOfPositions returns the total number of positions (1 for off + number of scenes).
func (s *goveeLightScene) GetNumberOfPositions(_ context.Context, _ map[string]interface{}) (uint32, []string, error) {
	return uint32(len(s.labels)), s.labels, nil
}
