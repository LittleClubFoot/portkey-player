package usb

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
)

const gadgetConfigDir = "/sys/kernel/config/usb_gadget/kidsmedia"

// GadgetManager handles USB mass storage gadget mode configuration.
type GadgetManager struct {
	mediaDir   string
	configPath string
	logger     *slog.Logger
}

// NewGadgetManager creates a manager for USB gadget mode.
func NewGadgetManager(mediaDir, configPath string, logger *slog.Logger) *GadgetManager {
	return &GadgetManager{
		mediaDir:   mediaDir,
		configPath: configPath,
		logger:     logger,
	}
}

// IsUSBPowered checks if the device is powered via USB (not wall adapter).
// On Raspberry Pi, this is heuristic-based on the max_current available.
func (g *GadgetManager) IsUSBPowered() bool {
	data, err := os.ReadFile("/sys/class/power_supply/usb/online")
	if err != nil {
		return false
	}
	return len(data) > 0 && data[0] == '1'
}

// Enable configures the Pi as a USB mass storage device exposing the media
// directory and config file. This requires the dwc2 and g_mass_storage modules.
func (g *GadgetManager) Enable() error {
	g.logger.Info("enabling USB gadget mode")

	// Load the required kernel modules.
	if err := loadModule("dwc2"); err != nil {
		return fmt.Errorf("loading dwc2 module: %w", err)
	}
	if err := loadModule("g_mass_storage", "file="+g.mediaDir, "stall=0", "removable=1"); err != nil {
		return fmt.Errorf("loading g_mass_storage module: %w", err)
	}

	g.logger.Info("USB gadget mode enabled", "media_dir", g.mediaDir)
	return nil
}

// Disable tears down USB gadget mode.
func (g *GadgetManager) Disable() error {
	g.logger.Info("disabling USB gadget mode")

	cmd := exec.Command("modprobe", "-r", "g_mass_storage")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("removing g_mass_storage module: %w", err)
	}

	return nil
}

func loadModule(name string, params ...string) error {
	args := append([]string{name}, params...)
	cmd := exec.Command("modprobe", args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("modprobe %s: %s: %w", name, string(output), err)
	}
	return nil
}
