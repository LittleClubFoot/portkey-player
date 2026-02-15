package storage

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"

	"github.com/LittleClubFoot/portkey-player/pkg/models"
)

// Mounter handles mounting and unmounting NAS shares.
type Mounter struct {
	logger *slog.Logger
}

// NewMounter creates a new NAS mounter.
func NewMounter(logger *slog.Logger) *Mounter {
	return &Mounter{logger: logger}
}

// MountAll mounts all configured NAS shares. Errors are logged but do not
// prevent the application from starting (graceful degradation).
func (m *Mounter) MountAll(shares []models.NASShare) []error {
	var errs []error
	for _, share := range shares {
		if err := m.Mount(share); err != nil {
			m.logger.Error("failed to mount share", "host", share.Host, "share", share.Share, "error", err)
			errs = append(errs, err)
		}
	}
	return errs
}

// Mount mounts a single NAS share.
func (m *Mounter) Mount(share models.NASShare) error {
	if err := os.MkdirAll(share.MountPoint, 0755); err != nil {
		return fmt.Errorf("creating mount point %s: %w", share.MountPoint, err)
	}

	// Check if already mounted.
	if isMounted(share.MountPoint) {
		m.logger.Info("share already mounted", "mount_point", share.MountPoint)
		return nil
	}

	switch share.Type {
	case "smb":
		return m.mountSMB(share)
	case "nfs":
		return m.mountNFS(share)
	default:
		return fmt.Errorf("unsupported share type: %s", share.Type)
	}
}

func (m *Mounter) mountSMB(share models.NASShare) error {
	source := fmt.Sprintf("//%s/%s", share.Host, share.Share)
	args := []string{
		"-t", "cifs",
		source, share.MountPoint,
		"-o", fmt.Sprintf("username=%s,password=%s,vers=3.0,iocharset=utf8", share.Username, share.Password),
	}

	m.logger.Info("mounting SMB share", "source", source, "target", share.MountPoint)
	cmd := exec.Command("mount", args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("mounting SMB share %s: %s: %w", source, string(output), err)
	}
	return nil
}

func (m *Mounter) mountNFS(share models.NASShare) error {
	source := fmt.Sprintf("%s:/%s", share.Host, share.Share)
	args := []string{
		"-t", "nfs",
		source, share.MountPoint,
		"-o", "nolock,soft,timeo=30",
	}

	m.logger.Info("mounting NFS share", "source", source, "target", share.MountPoint)
	cmd := exec.Command("mount", args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("mounting NFS share %s: %s: %w", source, string(output), err)
	}
	return nil
}

// UnmountAll unmounts all configured NAS shares.
func (m *Mounter) UnmountAll(shares []models.NASShare) {
	for _, share := range shares {
		if isMounted(share.MountPoint) {
			cmd := exec.Command("umount", share.MountPoint)
			if err := cmd.Run(); err != nil {
				m.logger.Warn("failed to unmount", "mount_point", share.MountPoint, "error", err)
			} else {
				m.logger.Info("unmounted share", "mount_point", share.MountPoint)
			}
		}
	}
}

func isMounted(path string) bool {
	cmd := exec.Command("mountpoint", "-q", path)
	return cmd.Run() == nil
}
