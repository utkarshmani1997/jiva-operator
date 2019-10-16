package driver

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/sirupsen/logrus"
	"k8s.io/kubernetes/pkg/util/mount"
)

// Mounter is an interface for mount operations
type Mounter interface {
	mount.Interface
	mount.Exec
	FormatAndMount(source string, target string, fstype string, options []string) error
	GetDeviceName(mountPath string) (string, int, error)
}

type NodeMounter struct {
	mount.SafeFormatAndMount
}

func newNodeMounter() Mounter {
	return &NodeMounter{
		mount.SafeFormatAndMount{
			Interface: mount.New(""),
			Exec:      mount.NewOsExec(),
		},
	}
}

func (m *NodeMounter) GetDeviceName(mountPath string) (string, int, error) {
	return mount.GetDeviceNameFromMount(m, mountPath)
}

func (m *NodeMounter) IsFormatted(source string) (bool, error) {
	if source == "" {
		return false, errors.New("source is not specified")
	}

	blkidCmd := "blkid"
	_, err := exec.LookPath(blkidCmd)
	if err != nil {
		if err == exec.ErrNotFound {
			return false, fmt.Errorf("%q executable not found in $PATH", blkidCmd)
		}
		return false, err
	}

	blkidArgs := []string{source}

	logrus.Infof("checking if source is formatted: cmd: %v args: %v", blkidCmd, blkidArgs)

	out, err := exec.Command(blkidCmd, blkidArgs...).CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("checking formatting failed: %v cmd: %q output: %q",
			err, blkidCmd, string(out))
	}

	if strings.TrimSpace(string(out)) == "" {
		return false, nil
	}

	return true, nil
}
