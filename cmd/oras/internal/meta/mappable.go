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

package meta

import (
	"reflect"
	"strings"
)

// ToMappable converts an reflect value into a map[string]any with json tags as
// key.
func ToMappable(v reflect.Value) any {
	switch v.Kind() {
	case reflect.Struct:
		t := v.Type()
		numField := t.NumField()
		ret := make(map[string]any)
		for i := 0; i < numField; i++ {
			fv := v.Field(i)
			tag := t.Field(i).Tag.Get("json")
			if tag == "" {
				continue
			}
			key, _, _ := strings.Cut(tag, ",")
			ret[key] = ToMappable(fv)
		}
		return ret
	case reflect.Slice, reflect.Array:
		ret := make([]any, v.Len())
		for i := 0; i < v.Len(); i++ {
			ret[i] = ToMappable(v.Index(i))
		}
		return ret
	case reflect.Ptr, reflect.Interface:
		if v.IsNil() {
			return nil
		}
		elem := ToMappable(v.Elem())
		return &elem
	}
	return v.Interface()
}