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

import ocispec "github.com/opencontainers/image-spec/specs-go/v1"

// DigestReference is a reference to an artifact with digest.
type DigestReference struct {
	Reference string `json:"reference"`
}

// ToDigestReference converts a name and digest to a digest reference.
func ToDigestReference(name string, digest string) DigestReference {
	return DigestReference{
		Reference: name + "@" + digest,
	}
}

// Descriptor is a descriptor with digest reference.
type Descriptor struct {
	DigestReference
	ocispec.Descriptor
}

// ToDescriptor converts a descriptor to a descriptor with digest reference.
func ToDescriptor(name string, desc ocispec.Descriptor) Descriptor {
	return Descriptor{
		DigestReference: ToDigestReference(name, desc.Digest.String()),
		Descriptor:      desc,
	}
}
