/*
Copyright The ORAS Authors.
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

package root

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

type demoOptions struct {
	pairs map[string]string
	slice []string
	tuple []string
}

func demoCmd() *cobra.Command {
	var demoOptions demoOptions
	cmd := &cobra.Command{
		Use:   "demo",
		Short: "Demo of flags",
		Args: func(cmd *cobra.Command, args []string) error {
			for i, arg := range args {
				fmt.Fprintf(os.Stderr, "[%d]: %s\n", i, arg)
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDemo(&demoOptions)
		},
	}

	cmd.Flags().StringToStringVarP(&demoOptions.pairs, "paired", "p", nil, "paired key-value pairs")
	cmd.Flags().StringSliceVarP(&demoOptions.slice, "slice", "s", []string{}, "Add a slice of values")
	cmd.Flags().StringArrayVarP(&demoOptions.tuple, "tuple", "t", []string{}, "Add a tuple of values")
	return cmd
}

func runDemo(o *demoOptions) error {
	fmt.Println("Key-value pairs:")
	for k, v := range o.pairs {
		fmt.Printf("> %s: %s\n", k, v)
	}
	fmt.Println("Slices:")
	for i, t := range o.slice {
		fmt.Printf("> [%d]: %s\n", i, t)
	}
	fmt.Println("Tuples:")
	for i, t := range o.tuple {
		fmt.Printf("> [%d]: %s\n", i, t)
	}
	return nil
}
