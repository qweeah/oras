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
	"errors"
	"net/http"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/errcode"
)

type PackFunc func(opts oras.PackOptions) (ocispec.Descriptor, error)
type CopyFunc func(desc ocispec.Descriptor) error

// Upload packs an oci artifact and copies it with oci image fallback.
func Upload(opts oras.PackOptions, pack PackFunc, copy CopyFunc, dst *remote.Repository) (ocispec.Descriptor, error) {
	root, err := pack(opts)
	if err != nil {
		return ocispec.Descriptor{}, err
	}

	if err := copy(root); opts.PackImageManifest || !isOciArtifactUnsupportedErr(err) {
		// no fallback
		return root, err
	}

	// fallback to OCI image
	dst.SetReferrersCapability(false)
	opts.PackImageManifest = true
	root, err = pack(opts)
	if err != nil {
		return ocispec.Descriptor{}, err
	}
	return root, copy(root)
}

func isOciArtifactUnsupportedErr(err error) bool {
	var errResp *errcode.ErrorResponse
	var errCode errcode.Error
	return errors.As(err, &errResp) && errResp.StatusCode == http.StatusBadRequest &&
		errors.As(errResp, &errCode) && errCode.Code == errcode.ErrorCodeManifestInvalid
}
