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

package tree

import (
	"fmt"
	"io"
	"strings"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"gopkg.in/yaml.v3"
	"oras.land/oras/cmd/oras/internal/display/metadata"
	"oras.land/oras/internal/registryutil"
	"oras.land/oras/internal/tree"
)

// discoverHandler handles json metadata output for discover events.
type discoverHandler struct {
	referrers registryutil.ReferrersFunc
	path      string
	desc      ocispec.Descriptor
	verbose   bool
	out       io.Writer
}

// OnDiscovered implements metadata.DiscoverHandler.
func (d *discoverHandler) OnDiscovered() error {
	root := tree.New(fmt.Sprintf("%s@%s", d.path, d.desc.Digest))
	err := d.fetchAllReferrers(d.desc, root)
	if err != nil {
		return err
	}
	return tree.Print(root)
}

func (d *discoverHandler) fetchAllReferrers(desc ocispec.Descriptor, node *tree.Node) error {
	results, err := d.referrers(desc)
	if err != nil {
		return err
	}

	for _, r := range results {
		// Find all indirect referrers
		referrerNode := node.AddPath(r.ArtifactType, r.Digest)
		if d.verbose {
			for k, v := range r.Annotations {
				bytes, err := yaml.Marshal(map[string]string{k: v})
				if err != nil {
					return err
				}
				referrerNode.AddPath(strings.TrimSpace(string(bytes)))
			}
		}
		err := d.fetchAllReferrers(
			ocispec.Descriptor{
				Digest:    r.Digest,
				Size:      r.Size,
				MediaType: r.MediaType,
			},
			referrerNode)
		if err != nil {
			return err
		}
	}
	return nil
}

// NewDiscoverHandler creates a new handler for discover events.
func NewDiscoverHandler(out io.Writer, path string, referrersFunc registryutil.ReferrersFunc, desc ocispec.Descriptor, verbose bool) metadata.DiscoverHandler {
	return &discoverHandler{
		path:      path,
		referrers: referrersFunc,
		desc:      desc,
		verbose:   verbose,
		out:       out,
	}
}
