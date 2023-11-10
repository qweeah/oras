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
	"encoding/json"
	"fmt"

	"github.com/spf13/pflag"
	"oras.land/oras/cmd/oras/internal/output/format"
)

// Format option struct.
type Format struct {
	format.Flag
}

func (opts *Format) ApplyFlags(fs *pflag.FlagSet) {
	fs.Var(&opts.Flag, "format-stdout", fmt.Sprintf("[Preview] summary output to stdout in the specified format (default %q)", format.Plain))
}

// Print prints the Output as JSON.
func (opts *Format) Print(data interface{}, prettify bool) error {
	var err error
	switch opts.Flag {
	case format.Json:
		var content []byte
		if prettify {
			content, err = json.MarshalIndent(data, "", "  ")
		} else {
			content, err = json.Marshal(data)
		}
		if err != nil {
			return err
		}
		_, err = fmt.Println(string(content))
		return err

	}
	return nil
}
