package common

import (
	"context"
	"fmt"

	govee "github.com/DTCurrie/govee-go"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/resource"
)

// Family is the model namespace and family for all viam-govee components.
var Family = resource.ModelNamespace("dtcurrie").WithFamily("viam-govee")

// ConnectToDevice creates a Govee API client and verifies the target device
// exists in the account. It returns the client and the matched Device.
func ConnectToDevice(ctx context.Context, apiKey, deviceID, sku string, logger logging.Logger) (*govee.Client, *govee.Device, error) {
	client := govee.New(apiKey)

	devices, err := client.GetDevices(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("govee: failed to list devices: %w", err)
	}

	for i, d := range devices {
		if d.DeviceID == deviceID && d.SKU == sku {
			logger.Debugf("connected to govee device %q (%s)", d.DeviceName, d.DeviceID)
			return client, &devices[i], nil
		}
	}

	return nil, nil, fmt.Errorf("govee: device %q (sku %s) not found in account", deviceID, sku)
}
