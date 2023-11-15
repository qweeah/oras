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
	"html/template"
	"io"

	"github.com/spf13/pflag"
)

type Format struct {
	template string
}

// ApplyFlag implements FlagProvider.ApplyFlag
func (opts *Format) ApplyFlags(fs *pflag.FlagSet) {
	fs.StringVar(&opts.template, "format", "", `Format output with go template syntax`)
}

func (opts *Format) WriteTo(w io.Writer, data interface{}) error {
	switch opts.template {
	case "json":
		// output json
		// write marshalled data
		b, err := json.Marshal(data)
		if err != nil {
			return err
		}
		_, err = w.Write(b)
		if err != nil {
			return err
		}
	default:
		// go templating
		var err error
		t := template.New("out") // todo: add sprig .Funcs(sprigFuncs)
		t, err = t.Parse(opts.template)
		if err != nil {
			return err
		}
		return t.Execute(w, data)
	}
	return nil
}
