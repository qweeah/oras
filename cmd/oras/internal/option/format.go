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
	"io"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/spf13/pflag"
)

type Format struct {
	Template string
}

// ApplyFlag implements FlagProvider.ApplyFlag
func (opts *Format) ApplyFlags(fs *pflag.FlagSet) {
	fs.StringVar(&opts.Template, "format", "", `Format output with go template syntax`)
}

func (opts *Format) WriteTo(w io.Writer, data interface{}) error {
	switch opts.Template {
	case "":
		return nil
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
		t := template.New("out").Funcs(sprig.FuncMap())
		t, err = t.Parse(opts.Template)
		if err != nil {
			return err
		}
		return t.Execute(w, data)
	}
	return nil
}
