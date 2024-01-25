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
	"testing"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

func TestToDescriptor(t *testing.T) {
	ociDesc := ocispec.Descriptor{
		MediaType:   "mocked.media.type",
		Digest:      "mocked-digest",
		Size:        123,
		URLs:        []string{"mocked-url"},
		Annotations: map[string]string{"mocked-annotation-key": "mocked-annotation-value"},
	}
	name := "mocked-name"
	cmp(t, ToDescriptor(name, ociDesc), ociDesc, name+"@"+ociDesc.Digest.String())

	ociDesc.Platform = &ocispec.Platform{
		Architecture: "mocked-arch",
		OS:           "mocked-os",
		Variant:      "mocked-variant",
	}
	cmp(t, ToDescriptor(name, ociDesc), ociDesc, name+"@"+ociDesc.Digest.String())
}

func cmp(t *testing.T, desc Descriptor, ociDesc ocispec.Descriptor, expectedRef string) {
	if desc.DigestReference.Ref != expectedRef {
		t.Errorf("expected digest reference %q, got %q", expectedRef, desc.DigestReference.Ref)
	}
	if desc.Size != ociDesc.Size {
		t.Errorf("expected size %d, got %d", ociDesc.Size, desc.Size)
	}
	if desc.MediaType != ociDesc.MediaType {
		t.Errorf("expected media type %q, got %q", ociDesc.MediaType, desc.MediaType)
	}
	if desc.Digest != ociDesc.Digest {
		t.Errorf("expected digest %q, got %q", ociDesc.Digest, desc.Digest)
	}
	if desc.Size != ociDesc.Size {
		t.Errorf("expected size %d, got %d", ociDesc.Size, desc.Size)
	}
	if !reflect.DeepEqual(desc.URLs, ociDesc.URLs) {
		t.Errorf("expected urls %v, got %v", ociDesc.URLs, desc.URLs)
	}
	if !reflect.DeepEqual(desc.Annotations, ociDesc.Annotations) {
		t.Errorf("expected annotations %v, got %v", ociDesc.Annotations, desc.Annotations)
	}
	if !reflect.DeepEqual(desc.Data, ociDesc.Data) {
		t.Errorf("expected data %v, got %v", ociDesc.Data, desc.Data)
	}
	if desc.ArtifactType != ociDesc.ArtifactType {
		t.Errorf("expected artifact type %q, got %q", ociDesc.ArtifactType, desc.ArtifactType)
	}
	// compare platform
	// 1. both nil, pass
	if desc.Platform == nil && ociDesc.Platform == nil {
		return
	}
	// 2. one nil, fail
	if desc.Platform == nil || ociDesc.Platform == nil {
		t.Errorf("expected platform %v, got %v", ociDesc.Platform, desc.Platform)
	}
	// 3. both not nil, compare
	if desc.Platform.Architecture != ociDesc.Platform.Architecture {
		t.Errorf("expected platform architecture %q, got %q", ociDesc.Platform.Architecture, desc.Platform.Architecture)
	}
	if desc.Platform.OS != ociDesc.Platform.OS {
		t.Errorf("expected platform os %q, got %q", ociDesc.Platform.OS, desc.Platform.OS)
	}
	if desc.Platform.Variant != ociDesc.Platform.Variant {
		t.Errorf("expected platform variant %q, got %q", ociDesc.Platform.Variant, desc.Platform.Variant)
	}

}
