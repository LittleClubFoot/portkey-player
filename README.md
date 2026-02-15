# portkey-player

A Go-based media player for Raspberry Pi Zero 2 W that plays videos and audio based on RFID/QR code scans. Designed to minimize screen time by letting children make intentional content choices through physical cards.

## Features

- **Scan-to-play**: RFID/QR tag scanning triggers immediate media playback
- **Physical controls**: 6 GPIO buttons for play/pause, stop, seek, and volume
- **Parental controls**: Quiet hours, daily play limits, bedtime content restrictions
- **NAS support**: Mount SMB/NFS network shares from Synology or other NAS devices
- **Playback logging**: SQLite-backed event log for parental review
- **USB gadget mode**: Present as mass storage device for easy content management
- **Headless operation**: No desktop environment required, boots straight to ready state

## Requirements

- Raspberry Pi Zero 2 W (or any ARM64 Pi)
- Raspberry Pi OS Lite (64-bit)
- mpv media player (`sudo apt install mpv`)
- USB RFID/QR scanner (HID keyboard device)
- Go 1.21+ (for building)

## Quick Start

```bash
# Build for Raspberry Pi
make build-pi

# Copy binary and configs to Pi, then:
sudo bash scripts/install.sh

# Edit configuration
sudo nano /opt/kidsmedia/config.json

# Start the service
sudo systemctl start kidsmedia
```

## Building

```bash
# Build for local machine
make build

# Build for Raspberry Pi (ARM64)
make build-pi

# Run tests
make test

# Run tests with coverage
make test-cover
```

## Configuration

Copy `configs/config.example.json` to `/opt/kidsmedia/config.json` and edit:

- **media**: Map tag IDs to media file paths and metadata
- **rules**: Set parental controls (quiet hours, daily limits, bedtime restrictions)
- **network**: Configure NAS share mounts
- **hardware**: Set GPIO pin mappings and player options

## GPIO Wiring

Default pin assignments (BCM numbering):

| Button      | GPIO Pin | Physical Pin |
|-------------|----------|-------------|
| Play/Pause  | 17       | 11          |
| Stop        | 27       | 13          |
| Rewind      | 22       | 15          |
| Forward     | 23       | 16          |
| Volume Up   | 24       | 18          |
| Volume Down | 25       | 22          |

Connect each button between the GPIO pin and GND. Internal pull-ups are used.

## Project Structure

```
cmd/player/         - Application entry point
internal/config/    - Configuration loading and validation
internal/input/     - Scanner and GPIO button handlers
internal/player/    - MPV player control via IPC
internal/resolver/  - Tag-to-media resolution
internal/rules/     - Parental control rule engine
internal/logger/    - SQLite playback event logging
internal/storage/   - NAS mounting and filesystem helpers
internal/usb/       - USB gadget mode management
pkg/models/         - Shared data models
configs/            - Example config and systemd service
scripts/            - Installation and setup scripts
```

## License

MIT License - see [LICENSE](LICENSE) for details.
