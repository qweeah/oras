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
	"unicode"
)

// ToMappable converts any type with 1 or more fields into a map[string]interface{}.
func ToMappable(in any) any {
	v := reflect.ValueOf(in)
	k := v.Kind()
	switch k {
	case reflect.Ptr, reflect.Interface:
		if v.IsNil() {
			return nil
		}
		return ToMappable(v.Elem().Interface())
	case reflect.Struct:
	default:
		return v
	}
	t := reflect.TypeOf(in)
	n := t.NumField()
	ret := make(map[string]any)

	for i := 0; i < n; i++ {
		v := v.Field(i)
		if !v.CanInterface() {
			continue
		}
		t := t.Field(i)
		rawName := t.Name
		lowedName := lowerFieldName(rawName)
		ret[lowedName] = ToMappable(v.Interface())
	}
	return ret
}

func lowerFieldName(name string) string {
	ret := ""
	i := 0
	for ; i < len(name); i++ {
		c := rune(name[i])
		if !unicode.IsUpper(c) {
			break
		}
		ret += string(unicode.ToLower(c))
	}
	return ret + name[i:]
}
