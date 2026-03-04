# Module viam-govee

A [Viam](https://www.viam.com/) module for controlling [Govee](https://us.govee.com/) smart lights using the [Govee Developer REST API](https://developer.govee.com/docs).

Mirrors the architecture of [viam-philips-hue](https://github.com/DTCurrie/viam-philips-hue) and is powered by the [govee-go](https://github.com/DTCurrie/govee-go) client library.

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

- A `govee-light-brightness` switch is emitted (for any device supporting `"turn"` or `"brightness"`).
- A `govee-light-sensor` is emitted (for retrievable devices only).
- Three `govee-light-color` switches are emitted for `red`, `green`, and `blue` channels (for devices supporting `"color"`).

If any color-capable devices are found, a single `govee-lights-mode` switch named `govee-mode` is emitted covering all of them.

## govee-light-brightness

Controls a single Govee device's on/off state and brightness. Implements the switch interface.

```json
{
  "api_key": "your-govee-api-key",
  "device": "AA:BB:CC:DD:EE:FF:00:11",
  "model": "H6159"
}
```

### Switch Positions

| Position | Effect                                 |
| -------- | -------------------------------------- |
| `0`      | Light off                              |
| `1`      | Light on at last-set brightness        |
| `2–100`  | Light on at that brightness percentage |

## govee-light-color

Controls a single RGB color channel on a Govee light that supports color. Implements the switch interface with a 0–255 range per channel.

```json
{
  "api_key": "your-govee-api-key",
  "device": "AA:BB:CC:DD:EE:FF:00:11",
  "model": "H6159",
  "channel": "red"
}
```

`channel` must be `"red"`, `"green"`, or `"blue"`. Use one component per channel. Each `SetPosition` call reads the current color state and modifies only the configured channel before sending the full RGB value.

### Switch Positions

| Position | Effect                                                  |
| -------- | ------------------------------------------------------- |
| `0–255`  | Channel intensity (maps 1:1 to the 0–255 channel value) |

## govee-light-sensor

Reports the current state of a single Govee light. Implements the sensor interface.

```json
{
  "api_key": "your-govee-api-key",
  "device": "AA:BB:CC:DD:EE:FF:00:11",
  "model": "H6159"
}
```

### Readings

**Device metadata** (always present):

| Key            | Type   | Description                                 |
| -------------- | ------ | ------------------------------------------- |
| `device_name`  | string | User-assigned device name                   |
| `model`        | string | Product model identifier                    |
| `device_id`    | string | MAC address                                 |
| `controllable` | bool   | Whether the device accepts control commands |
| `retrievable`  | bool   | Whether live state can be queried           |
| `support_cmds` | string | Comma-separated list of supported commands  |

**Live state** (present when `retrievable: true`):

| Key           | Type   | Description                                                             |
| ------------- | ------ | ----------------------------------------------------------------------- |
| `online`      | bool   | Whether the device is reachable (cached; may be stale)                  |
| `power_state` | string | `"on"` or `"off"`                                                       |
| `brightness`  | int    | Current brightness (0–100)                                              |
| `red`         | int    | Current red channel value (0–255)                                       |
| `green`       | int    | Current green channel value (0–255)                                     |
| `blue`        | int    | Current blue channel value (0–255)                                      |
| `color_temp`  | int    | Current color temperature in Kelvin (omitted if not in color temp mode) |

## govee-lights-mode

Controls pre-defined lighting modes across one or more devices. The default mode is `"none"`, which restores lights to their state before any mode was activated.

```json
{
  "api_key": "your-govee-api-key",
  "daylight": [
    { "device": "AA:BB:CC:DD:EE:FF:00:11", "model": "H6159" },
    { "device": "11:22:33:44:55:66:77:88", "model": "H6159" }
  ],
  "warm": [{ "device": "AA:BB:CC:DD:EE:FF:00:11", "model": "H6159" }]
}
```

Each mode key takes an array of device references. Only the modes you want to use need to be configured.

### Switch Positions

| Position | Mode       | Effect                                                        |
| -------- | ---------- | ------------------------------------------------------------- |
| `0`      | `none`     | Restore all devices to their saved pre-mode state             |
| `1`      | `daylight` | Cool daylight white (6500 K) at full brightness               |
| `2`      | `warm`     | Warm incandescent white (2700 K) at moderate brightness (60%) |

When switching to a mode, the current device state is automatically saved so it can be restored when returning to `none`.

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
      "name": "living-room-light",
      "api": "rdk:component:switch",
      "model": "dtcurrie:viam-govee:govee-light-brightness",
      "attributes": {
        "api_key": "your-govee-api-key",
        "device": "AA:BB:CC:DD:EE:FF:00:11",
        "model": "H6159"
      }
    },
    {
      "name": "living-room-light-red",
      "api": "rdk:component:switch",
      "model": "dtcurrie:viam-govee:govee-light-color",
      "attributes": {
        "api_key": "your-govee-api-key",
        "device": "AA:BB:CC:DD:EE:FF:00:11",
        "model": "H6159",
        "channel": "red"
      }
    },
    {
      "name": "living-room-light-sensor",
      "api": "rdk:component:sensor",
      "model": "dtcurrie:viam-govee:govee-light-sensor",
      "attributes": {
        "api_key": "your-govee-api-key",
        "device": "AA:BB:CC:DD:EE:FF:00:11",
        "model": "H6159"
      }
    },
    {
      "name": "govee-mode",
      "api": "rdk:component:switch",
      "model": "dtcurrie:viam-govee:govee-lights-mode",
      "attributes": {
        "api_key": "your-govee-api-key",
        "daylight": [{ "device": "AA:BB:CC:DD:EE:FF:00:11", "model": "H6159" }],
        "warm": [{ "device": "AA:BB:CC:DD:EE:FF:00:11", "model": "H6159" }]
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
