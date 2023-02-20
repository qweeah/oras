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

package option

import (
	"context"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"oras.land/oras/internal/trace"
)

// Common option struct.
type Common struct {
	debugFlag bool
	Verbose   bool

	Context func() context.Context
	Logger  logrus.FieldLogger
}

// ApplyFlags applies flags to a command flag set.
func (opts *Common) ApplyFlags(fs *pflag.FlagSet) {
	fs.BoolVarP(&opts.debugFlag, "debug", "d", false, "debug mode")
	fs.BoolVarP(&opts.Verbose, "verbose", "v", false, "verbose output")
}

// Parse sets up the logger and command context based on common options.
func (opts *Common) Parse(cmd *cobra.Command, _ []string) error {
	var logLevel logrus.Level
	if opts.debugFlag {
		logLevel = logrus.DebugLevel
	} else if opts.Verbose {
		logLevel = logrus.InfoLevel
	} else {
		logLevel = logrus.WarnLevel
	}
	var ctx context.Context
	ctx, opts.Logger = trace.WithLoggerLevel(cmd.Context(), logLevel)
	cmd.SetContext(ctx)
	opts.Context = cmd.Context
	return nil
}
