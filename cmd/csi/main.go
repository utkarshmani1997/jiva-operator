package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/utkarshmani1997/jiva-operator/pkg/config"
	"github.com/utkarshmani1997/jiva-operator/pkg/driver"
	"github.com/utkarshmani1997/jiva-operator/pkg/version"
)

/*
 * main routine to start the jiva-csi-driver. The same
 * binary is used for controller and agent deployment.
 * they both are differentiated via plugin command line
 * argument. To start the controller, we have to pass
 * --plugin=controller and to start it as agent, we have
 * to pass --plugin=agent.
 */
func main() {
	_ = flag.CommandLine.Parse([]string{})
	var config = config.Default()

	cmd := &cobra.Command{
		Use:   "jiva-csi-driver",
		Short: "driver for provisioning jiva volume",
		Long:  `provisions and deprovisions the volume`,
		Run: func(cmd *cobra.Command, args []string) {
			run(config)
		},
	}

	cmd.Flags().AddGoFlagSet(flag.CommandLine)

	cmd.PersistentFlags().StringVar(
		&config.NodeID, "nodeid", "node1", "NodeID to identify the node running this driver",
	)

	cmd.PersistentFlags().StringVar(
		&config.Version, "version", "", "Displays driver version",
	)

	cmd.PersistentFlags().StringVar(
		&config.Endpoint, "endpoint", "unix://csi/csi.sock", "CSI endpoint",
	)

	cmd.PersistentFlags().StringVar(
		&config.DriverName, "name", "jiva-csi-driver", "Name of this driver",
	)

	cmd.PersistentFlags().StringVar(
		&config.PluginType, "plugin", "jiva-csi-plugin", "Type of this driver i.e. controller or node",
	)

	err := cmd.Execute()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%s", err.Error())
		os.Exit(1)
	}
}

func run(config *config.Config) {
	if config.Version == "" {
		config.Version = version.Current()
	}

	logrus.Infof("%s - %s", version.Current(), version.GetGitCommit())
	logrus.Infof(
		"DriverName: %s Plugin: %s EndPoint: %s NodeID: %s",
		config.DriverName,
		config.PluginType,
		config.Endpoint,
		config.NodeID,
	)

	err := driver.New(config).Run()
	if err != nil {
		log.Fatalln(err)
	}
	os.Exit(0)
}