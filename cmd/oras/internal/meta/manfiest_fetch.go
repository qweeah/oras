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
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

type manifestFetch struct {
	DigestReference
	Config Descriptor   `json:"config"`
	Layers []Descriptor `json:"layers"`
}

// NewManifestFetch creates a new manifest fetch metadata for formatting.
func NewManifestFetch(path string, digest string, manifest ocispec.Manifest) manifestFetch {
	mf := manifestFetch{
		DigestReference: ToDigestReference(path, digest),
	}
	for _, layer := range manifest.Layers {
		mf.Layers = append(mf.Layers, ToDescriptor(path, layer))
	}
	return mf
}
