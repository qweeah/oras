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
	"io"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras/cmd/oras/internal/display/metadata"
	"oras.land/oras/cmd/oras/internal/display/metadata/model"
)

// discoverHandler handles json metadata output for discover events.
type discoverHandler struct {
	template  string
	path      string
	model     model.Discover
	recursive bool
	out       io.Writer
}

// NewDiscoverHandler creates a new handler for discover events.
func NewDiscoverHandler(out io.Writer, path string, root ocispec.Descriptor, recursive bool, template string) metadata.DiscoverHandler {
	return &discoverHandler{
		out:       out,
		model:     model.NewDiscover(path, root),
		path:      path,
		recursive: recursive,
		template:  template,
	}
}

// MultiLevelSupported implements metadata.DiscoverHandler.
func (h *discoverHandler) MultiLevelSupported() bool {
	return h.recursive
}

// OnDiscovered implements metadata.DiscoverHandler.
func (h *discoverHandler) OnDiscovered(referrer, subject ocispec.Descriptor) error {
	return h.model.Add(referrer, subject)
}

// OnCompleted implements metadata.DiscoverHandler.
func (h *discoverHandler) OnCompleted() error {
	return parseAndWrite(h.out, &h.model.Root, h.template)
}
