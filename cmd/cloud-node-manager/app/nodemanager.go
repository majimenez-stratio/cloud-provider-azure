/*
Copyright 2019 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package app

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apiserver/pkg/server"
	"k8s.io/apiserver/pkg/server/healthz"
	"k8s.io/apiserver/pkg/util/term"
	cloudnodeconfig "k8s.io/cloud-provider-azure/cmd/cloud-node-manager/app/config"
	"k8s.io/cloud-provider-azure/cmd/cloud-node-manager/app/options"
	nodeprovider "k8s.io/cloud-provider-azure/pkg/node"
	"k8s.io/cloud-provider-azure/pkg/nodemanager"
	"k8s.io/cloud-provider-azure/pkg/version"
	"k8s.io/cloud-provider-azure/pkg/version/verflag"
	cliflag "k8s.io/component-base/cli/flag"
	"k8s.io/component-base/cli/globalflag"
	"k8s.io/klog"
	genericcontrollermanager "k8s.io/kubernetes/cmd/controller-manager/app"
	utilflag "k8s.io/kubernetes/pkg/util/flag"
)

// NewCloudNodeManagerCommand creates a *cobra.Command object with default parameters
func NewCloudNodeManagerCommand() *cobra.Command {
	s, err := options.NewCloudNodeManagerOptions()
	if err != nil {
		klog.Fatalf("unable to initialize command options: %v", err)
	}

	cmd := &cobra.Command{
		Use:  "cloud-node-manager",
		Long: `The Cloud node manager is a daemon that reconciles node information for its running node.`,
		Run: func(cmd *cobra.Command, args []string) {
			verflag.PrintAndExitIfRequested("Cloud Node Manager")
			utilflag.PrintFlags(cmd.Flags())

			c, err := s.Config()
			if err != nil {
				fmt.Fprintf(os.Stderr, "%v\n", err)
				os.Exit(1)
			}

			if err := Run(c, wait.NeverStop); err != nil {
				fmt.Fprintf(os.Stderr, "%v\n", err)
				os.Exit(1)
			}

		},
	}

	fs := cmd.Flags()
	namedFlagSets := s.Flags()
	verflag.AddFlags(namedFlagSets.FlagSet("global"))
	globalflag.AddGlobalFlags(namedFlagSets.FlagSet("global"), cmd.Name())
	for _, f := range namedFlagSets.FlagSets {
		fs.AddFlagSet(f)
	}
	usageFmt := "Usage:\n  %s\n"
	cols, _, _ := term.TerminalSize(cmd.OutOrStdout())
	cmd.SetUsageFunc(func(cmd *cobra.Command) error {
		fmt.Fprintf(cmd.OutOrStderr(), usageFmt, cmd.UseLine())
		cliflag.PrintSections(cmd.OutOrStderr(), namedFlagSets, cols)
		return nil
	})
	cmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		fmt.Fprintf(cmd.OutOrStdout(), "%s\n\n"+usageFmt, cmd.Long, cmd.UseLine())
		cliflag.PrintSections(cmd.OutOrStdout(), namedFlagSets, cols)
	})

	return cmd
}

// Run runs the ExternalCMServer.  This should never exit.
func Run(c *cloudnodeconfig.Config, stopCh <-chan struct{}) error {
	// To help debugging, immediately log version
	klog.Infof("Version: %+v", version.Get())

	// Start the controller manager HTTP server
	var checks []healthz.HealthChecker
	if c.SecureServing != nil {
		unsecuredMux := genericcontrollermanager.NewBaseHandler(nil, checks...)
		handler := genericcontrollermanager.BuildHandlerChain(unsecuredMux, &c.Authorization, &c.Authentication)
		// TODO: handle stoppedCh returned by c.SecureServing.Serve
		if _, err := c.SecureServing.Serve(handler, 0, stopCh); err != nil {
			return err
		}
	}
	if c.InsecureServing != nil {
		unsecuredMux := genericcontrollermanager.NewBaseHandler(nil, checks...)
		insecureSuperuserAuthn := server.AuthenticationInfo{Authenticator: &server.InsecureSuperuser{}}
		handler := genericcontrollermanager.BuildHandlerChain(unsecuredMux, nil, &insecureSuperuserAuthn)
		if err := c.InsecureServing.Serve(handler, 0, stopCh); err != nil {
			return err
		}
	}

	run := func(ctx context.Context) {
		if err := startControllers(c, ctx.Done()); err != nil {
			klog.Fatalf("error running controllers: %v", err)
		}
	}

	run(context.TODO())
	panic("unreachable")
}

// startControllers starts the cloud specific controller loops.
func startControllers(c *cloudnodeconfig.Config, stopCh <-chan struct{}) error {
	klog.V(1).Infof("Starting cloud-node-manager...")

	// Start the CloudNodeController
	nodeController := nodemanager.NewCloudNodeController(
		c.NodeName,
		c.SharedInformers.Core().V1().Nodes(),
		// cloud node controller uses existing cluster role from node-controller
		c.ClientBuilder.ClientOrDie("node-controller"),
		nodeprovider.NewIMDSNodeProvider(),
		c.NodeStatusUpdateFrequency.Duration)

	go nodeController.Run(stopCh)

	klog.Infof("Started cloud-node-manager")

	// If apiserver is not running we should wait for some time and fail only then. This is particularly
	// important when we start node manager before apiserver starts.
	if err := genericcontrollermanager.WaitForAPIServer(c.VersionedClient, 10*time.Second); err != nil {
		klog.Fatalf("Failed to wait for apiserver being healthy: %v", err)
	}

	c.SharedInformers.Start(stopCh)

	select {}
}