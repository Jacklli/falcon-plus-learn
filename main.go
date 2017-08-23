// Copyright 2017 Xiaomi, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"fmt"
	"os"

	"github.com/open-falcon/falcon-plus/cmd"
	"github.com/spf13/cobra"  // Cobra is a library providing a simple interface to create powerful modern CLI interfaces similar to git & go tools.
)

var versionFlag bool

var RootCmd = &cobra.Command{
	Use: "open-falcon",
	RunE: func(c *cobra.Command, args []string) error {
		if versionFlag {
			fmt.Printf("Open-Falcon version %s, build %s\n", Version, GitCommit)
			return nil
		}
		return c.Usage()
	},
}

func init() {
	RootCmd.AddCommand(cmd.Start)
	RootCmd.AddCommand(cmd.Stop)
	RootCmd.AddCommand(cmd.Restart)
	RootCmd.AddCommand(cmd.Check)
	RootCmd.AddCommand(cmd.Monitor)
	RootCmd.AddCommand(cmd.Reload)

	RootCmd.Flags().BoolVarP(&versionFlag, "version", "v", false, "show version")
	cmd.Start.Flags().BoolVar(&cmd.PreqOrderFlag, "preq-order", false, "start modules in the order of prerequisites")
	cmd.Start.Flags().BoolVar(&cmd.ConsoleOutputFlag, "console-output", false, "print the module's output to the console")
}

func main() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}


/* ********** COBRA DEMO **********
package main

import (
        "fmt"
        "os"

        "github.com/spf13/cobra"
)

var versionFlag bool

var PreqOrderFlag bool
var ConsoleOutputFlag bool

var Start = &cobra.Command{
        Use:   "start [Module ...]",
        Short: "Start Open-Falcon modules",
        Long: `
                                Start the specified Open-Falcon modules and run until a stop command is received.
                                A module represents a single node in a cluster.
                                Modules:
                                        `,
        RunE: func(c *cobra.Command, args []string) error {
                fmt.Printf("Start with PreqOrderFlag: %v ConsoleOutputFlag: %v, args: %s\n", PreqOrderFlag, ConsoleOutputFlag, args)
                return nil
        },
        SilenceUsage:  true,
        SilenceErrors: true,
}

var RootCmd = &cobra.Command{
        Use: "open-falcon",
        RunE: func(c *cobra.Command, args []string) error {
                if versionFlag {
                        fmt.Printf("Open-Falcon version %d, build %d\n", 1, 2)
                        return nil
                }
                return c.Usage()
        },
}

func init() {
        RootCmd.AddCommand(Start)

        RootCmd.Flags().BoolVarP(&versionFlag, "version", "v", false, "show version")
        Start.Flags().BoolVar(&PreqOrderFlag, "preq-order", false, "start modules in the order of prerequisites")
        Start.Flags().BoolVar(&ConsoleOutputFlag, "console-output", false, "print the module's output to the console")
}

func main() {
        if err := RootCmd.Execute(); err != nil {
                fmt.Println(err)
                os.Exit(1)
        }
}
 ********** COBRA DEMO ********** */