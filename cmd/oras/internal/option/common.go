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
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"golang.org/x/term"
	"oras.land/oras/internal/trace"
)

// Common option struct.
type Common struct {
	Debug    bool
	Verbose  bool
	UseTTY   bool
	avoidTTY bool
}

// ApplyFlags applies flags to a command flag set.
func (opts *Common) ApplyFlags(fs *pflag.FlagSet) {
	fs.BoolVarP(&opts.Debug, "debug", "d", false, "debug mode")
	fs.BoolVarP(&opts.Verbose, "verbose", "v", false, "verbose output")
	fs.BoolVarP(&opts.avoidTTY, "noTTY", "", false, "[Preview] avoid using stdout as a terminal")
}

// WithContext returns a new FieldLogger and an associated Context derived from ctx.
func (opts *Common) WithContext(ctx context.Context) (context.Context, logrus.FieldLogger) {
	return trace.NewLogger(ctx, opts.Debug, opts.Verbose)
}

// Parse gets target options from user input.
func (opts *Common) Parse() error {
	if opts.avoidTTY {
		opts.UseTTY = false
	} else {
		opts.UseTTY = term.IsTerminal(int(os.Stdout.Fd()))
	}
	return nil
}
