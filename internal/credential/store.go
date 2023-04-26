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

package credential

import (
	credentials "github.com/oras-project/oras-credentials-go"
)

// NewStore generates a store based on the passed-in config file paths.
func NewStore(configPaths ...string) (credentials.Store, error) {
	opts := credentials.StoreOptions{AllowPlaintextPut: true}
	var store credentials.Store
	var err error
	if len(configPaths) == 0 {
		// use default docker config file path
		store, err = credentials.NewStoreFromDocker(opts)
	} else {
		var stores []credentials.Store
		for _, config := range configPaths {
			store, err := credentials.NewStore(config, opts)
			if err != nil {
				return nil, err
			}
			stores = append(stores, store)
		}
		store, err = credentials.NewStoreWithFallbacks(stores[0], stores[1:]...), nil
	}
	return store, err
}
