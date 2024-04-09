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

package template

import (
	"context"
	"io"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/registry"
	"oras.land/oras/cmd/oras/internal/display/metadata"
	"oras.land/oras/cmd/oras/internal/display/metadata/model"
)

// DiscoverHandler handles json metadata output for discover events.
type DiscoverHandler struct {
	ctx          context.Context
	repo         oras.ReadOnlyGraphTarget
	template     string
	path         string
	desc         ocispec.Descriptor
	artifactType string
	out          io.Writer
}

// OnDiscovered implements metadata.DiscoverHandler.
func (h *DiscoverHandler) OnDiscovered() error {
	refs, err := registry.Referrers(h.ctx, h.repo, h.desc, h.artifactType)
	if err != nil {
		return err
	}
	return parseAndWrite(h.out, model.NewDiscover(h.path, refs), h.template)
}

// NewDiscoverHandler creates a new handler for discover events.
func NewDiscoverHandler(ctx context.Context, out io.Writer, template string, path string, artifactType string, desc ocispec.Descriptor, repo oras.ReadOnlyGraphTarget) metadata.DiscoverHandler {
	return &DiscoverHandler{
		template:     template,
		path:         path,
		ctx:          ctx,
		repo:         repo,
		desc:         desc,
		artifactType: artifactType,
		out:          out,
	}
}
