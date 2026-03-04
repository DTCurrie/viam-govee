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

// GoveeLightDIYScene is the model identifier for the govee-light-diy-scene component.
var GoveeLightDIYScene = common.Family.WithModel("govee-light-diy-scene")

func init() {
	resource.RegisterComponent(toggleswitch.API, GoveeLightDIYScene,
		resource.Registration[toggleswitch.Switch, *DIYSceneConfig]{
			Constructor: newGoveeLightDIYScene,
		},
	)
}

// DIYSceneConfig is the configuration for the govee-light-diy-scene component.
type DIYSceneConfig struct {
	APIKey   string `json:"api_key"`
	DeviceID string `json:"device"`
	SKU      string `json:"sku"`
}

// Validate checks that all required fields are set.
func (cfg *DIYSceneConfig) Validate(_ string) ([]string, []string, error) {
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

type goveeLightDIYScene struct {
	resource.AlwaysRebuild
	resource.TriviallyCloseable

	name   resource.Name
	logger logging.Logger
	cfg    *DIYSceneConfig

	client    *govee.Client
	diyScenes []govee.DIYSceneOption // cached at construction time
	labels    []string               // position labels: ["off", scene1.Name, ...]

	mu       sync.Mutex
	position uint32
}

func newGoveeLightDIYScene(ctx context.Context, _ resource.Dependencies, rawConf resource.Config, logger logging.Logger) (toggleswitch.Switch, error) {
	conf, err := resource.NativeConfig[*DIYSceneConfig](rawConf)
	if err != nil {
		return nil, err
	}

	client, _, err := common.ConnectToDevice(ctx, conf.APIKey, conf.DeviceID, conf.SKU, logger)
	if err != nil {
		return nil, err
	}

	diyScenes, err := client.GetDIYScenes(ctx, conf.SKU, conf.DeviceID)
	if err != nil {
		return nil, fmt.Errorf("govee: failed to fetch DIY scenes for %s: %w", conf.DeviceID, err)
	}

	labels := make([]string, 0, len(diyScenes)+1)
	labels = append(labels, "off")
	for _, sc := range diyScenes {
		labels = append(labels, sc.Name)
	}

	return &goveeLightDIYScene{
		name:      rawConf.ResourceName(),
		logger:    logger,
		cfg:       conf,
		client:    client,
		diyScenes: diyScenes,
		labels:    labels,
	}, nil
}

func (s *goveeLightDIYScene) Name() resource.Name {
	return s.name
}

func (s *goveeLightDIYScene) DoCommand(_ context.Context, _ map[string]interface{}) (map[string]interface{}, error) {
	return nil, nil
}

// SetPosition activates a user-created DIY scene by its position index.
// Position 0 turns the device off. Positions 1..N activate the corresponding
// DIY scene (in the order reported by the Govee API at startup).
func (s *goveeLightDIYScene) SetPosition(ctx context.Context, position uint32, _ map[string]interface{}) error {
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

	scene := s.diyScenes[position-1]

	if err := s.client.TurnOn(ctx, s.cfg.SKU, s.cfg.DeviceID); err != nil {
		return fmt.Errorf("govee: failed to turn on device: %w", err)
	}

	if err := s.client.SetDIYScene(ctx, s.cfg.SKU, s.cfg.DeviceID, scene.Value); err != nil {
		return fmt.Errorf("govee: failed to set DIY scene %q: %w", scene.Name, err)
	}

	s.mu.Lock()
	s.position = position
	s.mu.Unlock()
	return nil
}

// GetPosition returns the last position set via SetPosition.
func (s *goveeLightDIYScene) GetPosition(_ context.Context, _ map[string]interface{}) (uint32, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.position, nil
}

// GetNumberOfPositions returns the total number of positions (1 for off + number of DIY scenes).
func (s *goveeLightDIYScene) GetNumberOfPositions(_ context.Context, _ map[string]interface{}) (uint32, []string, error) {
	return uint32(len(s.labels)), s.labels, nil
}
