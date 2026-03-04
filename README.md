> [!WARNING]
> **Work in Progress** — This module is under active development and is not guaranteed to work. APIs, configuration, and behavior may change without notice. Use at your own risk.

# Module viam-govee

A [Viam](https://www.viam.com/) module for controlling [Govee](https://us.govee.com/) smart lights using the [Govee OpenAPI](https://developer.govee.com/docs).

Powered by the [govee-go](https://github.com/DTCurrie/govee-go) client library.

[Viam Registry](https://app.viam.com/module/dtcurrie/viam-govee)

## Setup

### Getting a Govee API Key

1. Open the Govee Home app on your phone.
2. Go to **Profile → Settings → Apply for API Key**.
3. Submit your information and wait for the key to arrive by email.

Only Wi-Fi–connected devices are supported by the Govee API. Bluetooth-only devices cannot be controlled.

## govee-discovery

Discovery service that finds all Govee devices in your account and emits ready-to-use Viam resource configurations for each one.

```json
{
  "api_key": "your-govee-api-key"
}
```

For each device discovered:

**Smart plugs** (devices with type `devices.types.socket`):

- A `govee-plug-switch` switch is emitted.
- A `govee-plug-sensor` sensor is emitted.

**Smart lights** (all other devices):

- A `govee-light-switch` switch is emitted for devices supporting `on_off` (`powerSwitch`).
- A `govee-light-brightness` switch is emitted for devices supporting `brightness`.
- A `govee-light-color-temp` switch is emitted for devices supporting color temperature (`colorTemperatureK`).
- A `govee-light-sensor` sensor is emitted for every light device.
- Three `govee-light-color` switches (`red`, `green`, `blue`) are emitted for devices supporting `colorRgb`.
- A `govee-light-scene` switch is emitted for devices supporting dynamic scenes (`lightScene`).
- A `govee-light-diy-scene` switch is emitted for devices supporting user-created DIY scenes (`diyScene`).
- A `govee-light-snapshot` switch is emitted for devices supporting saved snapshots (`snapshot`).

## govee-light-switch

Controls a single Govee light's power state. Implements the switch interface.

```json
{
  "api_key": "your-govee-api-key",
  "device": "AA:BB:CC:DD:EE:FF:00:11",
  "sku": "H6159"
}
```

### Switch Positions

| Position | Effect    |
| -------- | --------- |
| `0`      | Light off |
| `1`      | Light on  |

`GetPosition` queries live state from the API and falls back to the last locally tracked position if the device is unreachable.

## govee-light-brightness

Controls a single Govee light's brightness. Implements the switch interface.

```json
{
  "api_key": "your-govee-api-key",
  "device": "AA:BB:CC:DD:EE:FF:00:11",
  "sku": "H6159"
}
```

### Switch Positions

| Position | Effect                                             |
| -------- | -------------------------------------------------- |
| `1–100`  | Brightness percentage (maps 1:1 to API brightness) |

Use `govee-light-switch` to turn the device on or off.

## govee-light-color-temp

Controls a single Govee light's color temperature. Implements the switch interface.

```json
{
  "api_key": "your-govee-api-key",
  "device": "AA:BB:CC:DD:EE:FF:00:11",
  "sku": "H6159"
}
```

The Kelvin range (e.g. 2000–9000 K) is read from the device's `colorTemperatureK` capability at startup and mapped linearly across positions 1–100.

### Switch Positions

| Position | Effect                                                       |
| -------- | ------------------------------------------------------------ |
| `1`      | Warmest color temperature (e.g. 2000 K)                      |
| `1–100`  | Linear range from warmest to coolest (e.g. 2000 K to 9000 K) |
| `100`    | Coolest color temperature (e.g. 9000 K)                      |

Use `govee-light-switch` to turn the device on or off.

## govee-light-color

Controls a single RGB color channel on a Govee light that supports color. Implements the switch interface with a 0–255 range per channel.

```json
{
  "api_key": "your-govee-api-key",
  "device": "AA:BB:CC:DD:EE:FF:00:11",
  "sku": "H6159",
  "channel": "red"
}
```

`channel` must be `"red"`, `"green"`, or `"blue"`. Use one component per channel. Each `SetPosition` call reads the current color state and modifies only the configured channel before sending the full RGB value.

### Switch Positions

| Position | Effect                                                  |
| -------- | ------------------------------------------------------- |
| `0–255`  | Channel intensity (maps 1:1 to the 0–255 channel value) |

## govee-light-scene

Activates a dynamic scene on a single Govee light. Scenes are fetched from the Govee API at startup and exposed as numbered positions. Position `0` turns the device off; all other positions activate the corresponding scene.

```json
{
  "api_key": "your-govee-api-key",
  "device": "AA:BB:CC:DD:EE:FF:00:11",
  "sku": "H6159"
}
```

### Switch Positions

| Position | Effect                               |
| -------- | ------------------------------------ |
| `0`      | Light off                            |
| `1`      | First dynamic scene (e.g. "Sunrise") |
| `2`      | Second dynamic scene (e.g. "Ocean")  |
| `…`      | Additional scenes in API order       |

Use `GetNumberOfPositions` to retrieve the full list of available scene names for a device.

## govee-light-diy-scene

Activates a user-created DIY scene on a single Govee light. DIY scenes are scenes you have built in the Govee Home app. They are fetched from the Govee API at startup and exposed as numbered positions. Position `0` turns the device off; all other positions activate the corresponding DIY scene.

```json
{
  "api_key": "your-govee-api-key",
  "device": "AA:BB:CC:DD:EE:FF:00:11",
  "sku": "H6159"
}
```

### Switch Positions

| Position | Effect                                   |
| -------- | ---------------------------------------- |
| `0`      | Light off                                |
| `1`      | First DIY scene (e.g. "My Custom Scene") |
| `2`      | Second DIY scene (e.g. "Party Mode")     |
| `…`      | Additional DIY scenes in API order       |

Use `GetNumberOfPositions` to retrieve the full list of available DIY scene names for a device.

## govee-light-snapshot

Activates a saved snapshot on a single Govee light. Snapshots are color and scene configurations you have saved in the Govee Home app. They are parsed from the device's static capabilities at startup and exposed as numbered positions. Position `0` turns the device off; all other positions activate the corresponding snapshot.

```json
{
  "api_key": "your-govee-api-key",
  "device": "AA:BB:CC:DD:EE:FF:00:11",
  "sku": "H6159"
}
```

### Switch Positions

| Position | Effect                                   |
| -------- | ---------------------------------------- |
| `0`      | Light off                                |
| `1`      | First snapshot (e.g. "Sunrise")          |
| `2`      | Second snapshot (e.g. "Sunset")          |
| `…`      | Additional snapshots in capability order |

Use `GetNumberOfPositions` to retrieve the full list of available snapshot names for a device.

## govee-light-sensor

Reports the current state of a single Govee light. Implements the sensor interface.

```json
{
  "api_key": "your-govee-api-key",
  "device": "AA:BB:CC:DD:EE:FF:00:11",
  "sku": "H6159"
}
```

### Readings

**Device metadata** (always present):

| Key            | Type     | Description                                      |
| -------------- | -------- | ------------------------------------------------ |
| `device_name`  | string   | User-assigned device name                        |
| `sku`          | string   | Product model identifier (SKU)                   |
| `device_id`    | string   | MAC address                                      |
| `device_type`  | string   | Device type (e.g. `devices.types.light`)         |
| `capabilities` | []string | List of supported capability type/instance pairs |

**Live state** (attempted on every call; absent if unavailable):

| Key                 | Type   | Description                                                                                                                                                 |
| ------------------- | ------ | ----------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `online`            | bool   | Whether the device is reachable                                                                                                                             |
| `power_state`       | bool   | `true` if on, `false` if off                                                                                                                                |
| `brightness`        | int    | Current brightness (0–100)                                                                                                                                  |
| `red`               | int    | Current red channel value (0–255)                                                                                                                           |
| `green`             | int    | Current green channel value (0–255)                                                                                                                         |
| `blue`              | int    | Current blue channel value (0–255)                                                                                                                          |
| `color_temp`        | int    | Current color temperature in Kelvin (omitted if not in color temp mode)                                                                                     |
| `toggle_<instance>` | bool   | State of each toggle capability the device reports (e.g. `toggle_gradientToggle`). Only present when the device has that toggle and its state is available. |
| `note`              | string | Error message if live state could not be retrieved                                                                                                          |

## govee-plug-switch

Controls a single Govee smart plug's power state. Implements the switch interface.

```json
{
  "api_key": "your-govee-api-key",
  "device": "11:22:33:44:55:66:77:88",
  "sku": "H5080"
}
```

### Switch Positions

| Position | Effect   |
| -------- | -------- |
| `0`      | Plug off |
| `1`      | Plug on  |

`GetPosition` queries live state from the API and falls back to the last locally tracked position if the device is unreachable.

## govee-plug-sensor

Reports the current state of a single Govee smart plug. Implements the sensor interface.

```json
{
  "api_key": "your-govee-api-key",
  "device": "11:22:33:44:55:66:77:88",
  "sku": "H5080"
}
```

### Readings

**Device metadata** (always present):

| Key            | Type     | Description                                      |
| -------------- | -------- | ------------------------------------------------ |
| `device_name`  | string   | User-assigned device name                        |
| `sku`          | string   | Product model identifier (SKU)                   |
| `device_id`    | string   | MAC address                                      |
| `device_type`  | string   | Device type (e.g. `devices.types.socket`)        |
| `capabilities` | []string | List of supported capability type/instance pairs |

**Live state** (attempted on every call; absent if unavailable):

| Key                 | Type   | Description                                                                                                                                                |
| ------------------- | ------ | ---------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `online`            | bool   | Whether the device is reachable                                                                                                                            |
| `power_state`       | bool   | `true` if on, `false` if off                                                                                                                               |
| `toggle_<instance>` | bool   | State of each toggle capability the device reports (e.g. `toggle_socketToggle1`). Only present when the device has that toggle and its state is available. |
| `note`              | string | Error message if live state could not be retrieved                                                                                                         |

## Viam Config Example

```json
{
  "services": [
    {
      "name": "govee-discovery",
      "api": "rdk:service:discovery",
      "model": "dtcurrie:viam-govee:govee-discovery",
      "attributes": {
        "api_key": "your-govee-api-key"
      }
    }
  ],
  "components": [
    {
      "name": "living-room-light-switch",
      "api": "rdk:component:switch",
      "model": "dtcurrie:viam-govee:govee-light-switch",
      "attributes": {
        "api_key": "your-govee-api-key",
        "device": "AA:BB:CC:DD:EE:FF:00:11",
        "sku": "H6159"
      }
    },
    {
      "name": "living-room-light",
      "api": "rdk:component:switch",
      "model": "dtcurrie:viam-govee:govee-light-brightness",
      "attributes": {
        "api_key": "your-govee-api-key",
        "device": "AA:BB:CC:DD:EE:FF:00:11",
        "sku": "H6159"
      }
    },
    {
      "name": "living-room-light-red",
      "api": "rdk:component:switch",
      "model": "dtcurrie:viam-govee:govee-light-color",
      "attributes": {
        "api_key": "your-govee-api-key",
        "device": "AA:BB:CC:DD:EE:FF:00:11",
        "sku": "H6159",
        "channel": "red"
      }
    },
    {
      "name": "living-room-light-scene",
      "api": "rdk:component:switch",
      "model": "dtcurrie:viam-govee:govee-light-scene",
      "attributes": {
        "api_key": "your-govee-api-key",
        "device": "AA:BB:CC:DD:EE:FF:00:11",
        "sku": "H6159"
      }
    },
    {
      "name": "living-room-light-diy-scene",
      "api": "rdk:component:switch",
      "model": "dtcurrie:viam-govee:govee-light-diy-scene",
      "attributes": {
        "api_key": "your-govee-api-key",
        "device": "AA:BB:CC:DD:EE:FF:00:11",
        "sku": "H6159"
      }
    },
    {
      "name": "living-room-light-sensor",
      "api": "rdk:component:sensor",
      "model": "dtcurrie:viam-govee:govee-light-sensor",
      "attributes": {
        "api_key": "your-govee-api-key",
        "device": "AA:BB:CC:DD:EE:FF:00:11",
        "sku": "H6159"
      }
    },
    {
      "name": "office-plug",
      "api": "rdk:component:switch",
      "model": "dtcurrie:viam-govee:govee-plug-switch",
      "attributes": {
        "api_key": "your-govee-api-key",
        "device": "11:22:33:44:55:66:77:88",
        "sku": "H5080"
      }
    },
    {
      "name": "office-plug-sensor",
      "api": "rdk:component:sensor",
      "model": "dtcurrie:viam-govee:govee-plug-sensor",
      "attributes": {
        "api_key": "your-govee-api-key",
        "device": "11:22:33:44:55:66:77:88",
        "sku": "H5080"
      }
    }
  ]
}
```

## Building

```shell
# Build the module binary
make

# Run tests
make test

# Build the deployable tarball
make module
```

## Rate Limits

The Govee API enforces strict rate limits. When multiple components share the same device, requests can accumulate quickly:

| Scope                         | Limit               |
| ----------------------------- | ------------------- |
| All APIs combined             | 10,000 requests/day |
| `GetDevices`                  | 10 requests/minute  |
| `GetDeviceState` (per device) | 10 requests/minute  |
| Control commands (per device) | 10 requests/minute  |

Avoid polling sensors faster than once every 6 seconds per device to stay within the per-device state limit.

## License

MIT — see [LICENSE](LICENSE).
