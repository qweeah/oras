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

package oci

import (
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
)

var ErrArtifactUnsupported error

type PackFunc func(opts oras.PackOptions) (ocispec.Descriptor, error)
type CopyFunc func(desc ocispec.Descriptor) error

// PackAndCopy.
func PackAndCopy(opts oras.PackOptions, pack PackFunc, copy CopyFunc) (ocispec.Descriptor, error) {
	// try OCI artifact first
	opts.PackImageManifest = false
	root, err := pack(opts)
	if err != nil {
		return ocispec.Descriptor{}, err
	}

	// if err = copy(root); errors.Is(err, ErrArtifactUnsupported) {
	if err = copy(root); err != nil {
		// fallback to OCI image
		opts.PackImageManifest = true
		root, err = pack(opts)
		if err != nil {
			return ocispec.Descriptor{}, err
		}
		err = copy(root)
	}
	return root, err
}
