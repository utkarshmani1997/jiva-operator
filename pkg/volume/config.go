package volume

/*
import (
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/container-storage-interface/spec/lib/go/csi"
)

var (
	Resources = []string{"replicaMaxCPU", "replicaMinCPU", "replicaMaxMemory", "replicaMinMemory", "targetMaxCPU", "targetMinCPU", "targetMaxMemory", "targetMinMemory", "replicaCount"}
)

type Config struct {
	Name         string
	ReplicaCount string
	Jiva
}

type Jiva struct {
	Replica
	Controller
}

type Resource struct {
	CPU
	Memory
}

type Replica struct {
	Resource
}

type Controller struct {
	Resource
}

type CPU struct {
	Limits
}

type Memory struct {
	Limits
}

type Limits struct {
	Max string
	Min string
}

func (c *Config) setCPULimits(res, limit string) {
	if strings.HasPrefix(res, "targetMax") {
		c.Jiva.Controller.CPU.Max = limit
	} else if strings.HasPrefix(res, "targetMin") {
		c.Jiva.Controller.CPU.Min = limit
	} else if strings.HasPrefix(res, "replicaMax") {
		c.Jiva.Replica.CPU.Max = limit
	} else if strings.HasPrefix(res, "replicaMin") {
		c.Jiva.Replica.CPU.Min = limit
	}
}

func (c *Config) setMemoryLimits(res, limit string) {
	if strings.HasPrefix(res, "targetMax") {
		c.Jiva.Controller.Memory.Max = limit
	} else if strings.HasPrefix(res, "targetMin") {
		c.Jiva.Controller.Memory.Min = limit
	} else if strings.HasPrefix(res, "replicaMax") {
		c.Jiva.Replica.Memory.Max = limit
	} else if strings.HasPrefix(res, "replicaMin") {
		c.Jiva.Replica.Memory.Min = limit
	}
}

type ResourceParameters func(param string) (string, bool)

func HasResourceParameters(req *csi.CreateVolumeRequest) ResourceParameters {
	return func(param string) (string, bool) {
		if val, ok := req.GetParameters()[param]; !ok {
			return "", false
		} else {
			return val, true
		}
	}
}

func (c *Config) validateConfig(req *csi.CreateVolumeRequest) []error {
	var errs []error
	for _, res := range Resources {
		if val, ok := HasResourceParameters(req)(res); !ok {
			errs = append(errs, fmt.Errorf("missing resource: %v", res))
		} else {
			if strings.HasSuffix(res, "Memory") {
				c.setMemoryLimits(res, val)
			} else if strings.HasSuffix(res, "CPU") {
				c.setCPULimits(res, val)
			} else if res == "replicaCount" {
				c.ReplicaCount = val
			}
		}
	}

	return errs
}

func NewConfig(req *csi.CreateVolumeRequest) Config {
	c := Config{
		Name: req.GetName(),
	}

	if errs := c.validateConfig(req); errs != nil {
		logrus.Warnf("Missing configs: %v, proceed with default settings", errs)
	}

	if c.ReplicaCount == "" {
		c.ReplicaCount = "3"
	}
	return c
}
*/
