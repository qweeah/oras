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
	"errors"
	"os"

	"github.com/spf13/pflag"
	"golang.org/x/term"
)

// Terminal output option struct.
type TTY struct {
	IsTTY bool
}

// ApplyFlags applies flags to a command flag set.
func (opts *TTY) ApplyFlags(fs *pflag.FlagSet) {
	fs.BoolVarP(&opts.IsTTY, "isTTY", "", false, "[Preview] will output TTY if set to true")
}

// Parse gets target options from user input.
func (opts *TTY) Parse() error {
	if opts.IsTTY && !term.IsTerminal(int(os.Stdout.Fd())) {
		return errors.New("isTTY is set to true but stdout is not a TTY")
	}
	return nil
}
