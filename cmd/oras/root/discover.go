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

package root

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/opencontainers/image-spec/specs-go"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/registry"
	"oras.land/oras/cmd/oras/internal/argument"
	"oras.land/oras/cmd/oras/internal/display"
	"oras.land/oras/cmd/oras/internal/errors"
	oerrors "oras.land/oras/cmd/oras/internal/errors"
	"oras.land/oras/cmd/oras/internal/option"
	"oras.land/oras/internal/tree"
)

type discoverOptions struct {
	option.Common
	option.Platform
	option.Target
	option.Format

	artifactType string
	outputType   string
}

func discoverCmd() *cobra.Command {
	var opts discoverOptions
	cmd := &cobra.Command{
		Use:   "discover [flags] <name>{:<tag>|@<digest>}",
		Short: "[Preview] Discover referrers of a manifest in a registry or an OCI image layout",
		Long: `[Preview] Discover referrers of a manifest in a registry or an OCI image layout

** This command is in preview and under development. **

Example - Discover direct referrers of manifest 'hello:v1' in registry 'localhost:5000':
  oras discover localhost:5000/hello:v1

Example - Discover direct referrers via referrers API:
  oras discover --distribution-spec v1.1-referrers-api localhost:5000/hello:v1

Example - Discover direct referrers via tag scheme:
  oras discover --distribution-spec v1.1-referrers-tag localhost:5000/hello:v1

Example - Discover all the referrers of manifest 'hello:v1' in registry 'localhost:5000', displayed in a tree view:
  oras discover -o tree localhost:5000/hello:v1

Example - Discover all the referrers of manifest with annotations, displayed in a tree view:
  oras discover -v -o tree localhost:5000/hello:v1

Example - Discover referrers with type 'test-artifact' of manifest 'hello:v1' in registry 'localhost:5000':
  oras discover --artifact-type test-artifact localhost:5000/hello:v1

Example - Discover referrers of the manifest tagged 'v1' in an OCI image layout folder 'layout-dir':
  oras discover --oci-layout layout-dir:v1
  oras discover --oci-layout -v -o tree layout-dir:v1
`,
		Args: oerrors.CheckArgs(argument.Exactly(1), "the target artifact to discover referrers from"),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if opts.outputType != "" && opts.Template != "" {
				return fmt.Errorf("cannot use both --output and --format")
			}
			if opts.outputType == "" && opts.Template == "" {
				opts.Template = "tree"
			}
			opts.RawReference = args[0]
			return option.Parse(&opts)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			display.Set(opts.Template, opts.TTY)
			return runDiscover(cmd.Context(), opts)
		},
	}

	cmd.Flags().StringVarP(&opts.artifactType, "artifact-type", "", "", "artifact type")
	cmd.Flags().StringVarP(&opts.outputType, "output", "o", "", "[Deprecated] format in which to display referrers (table, json, or tree). tree format will also show indirect referrers")
	cmd.Flags().StringVar(&opts.Template, "format", "", `Format output using a custom template:
'tree':       Get referrers recursively and print in tree format (default)
'table':      Get direct referrers and output in table format
'json':       Get direct referrers and output in JSON format
'$TEMPLATE':  Print output using the given Go template.`)
	opts.EnableDistributionSpecFlag()
	option.ApplyFlags(&opts, cmd.Flags())
	return cmd
}

func runDiscover(ctx context.Context, opts discoverOptions) error {
	ctx, logger := opts.WithContext(ctx)
	repo, err := opts.NewReadonlyTarget(ctx, opts.Common, logger)
	if err != nil {
		return err
	}
	if err := opts.EnsureReferenceNotEmpty(); err != nil {
		return err
	}

	// discover artifacts
	resolveOpts := oras.DefaultResolveOptions
	resolveOpts.TargetPlatform = opts.Platform.Platform
	desc, err := oras.Resolve(ctx, repo, opts.Reference, resolveOpts)
	if err != nil {
		return err
	}

	if opts.Template != "" {
		if err := output(ctx, opts.Template, opts, desc, repo); err != errors.ErrInvalidOutputType {
			// done or unexpected error
			return err
		}
		// formatting as index
		refs, err := graph.Referrers(ctx, repo, desc, opts.artifactType)
		if err != nil {
			return err
		}
		return opts.WriteTo(os.Stdout, ocispec.Index{
			Versioned: specs.Versioned{
				SchemaVersion: 2, // historical value. does not pertain to OCI or docker version
			},
			MediaType: ocispec.MediaTypeImageIndex,
			Manifests: refs,
		})
	}
	fmt.Fprintf(os.Stderr, "[DEPRECATED] --output is deprecated, try `--format %s` instead\n", opts.outputType)
	return output(ctx, opts.Template, opts, desc, repo)
}

func output(ctx context.Context, outputType string, opts discoverOptions, desc ocispec.Descriptor, repo option.ReadOnlyGraphTagFinderTarget) error {
	if outputType == "tree" || outputType == "" {
		// default to tree output
		root := tree.New(fmt.Sprintf("%s@%s", opts.Path, desc.Digest))
		err := fetchAllReferrers(ctx, repo, desc, opts.artifactType, root, &opts)
		if err != nil {
			return err
		}
		return tree.Print(root)
	}

	switch outputType {
	case "table", "json":
		refs, err := registry.Referrers(ctx, repo, desc, opts.artifactType)
		if err != nil {
			return err
		}
		if outputType == "json" {
			return printDiscoveredReferrersJSON(desc, refs)
		}

		if outputType == "table" {
			if n := len(refs); n > 1 {
				fmt.Println("Discovered", n, "artifacts referencing", opts.Reference)
			} else {
				fmt.Println("Discovered", n, "artifact referencing", opts.Reference)
			}
			fmt.Println("Digest:", desc.Digest)
			if len(refs) > 0 {
				fmt.Println()
				return printDiscoveredReferrersTable(refs, opts.Verbose)
			}
		}
	}
	return oerrors.ErrInvalidOutputType
}

func fetchAllReferrers(ctx context.Context, repo oras.ReadOnlyGraphTarget, desc ocispec.Descriptor, artifactType string, node *tree.Node, opts *discoverOptions) error {
	results, err := registry.Referrers(ctx, repo, desc, artifactType)
	if err != nil {
		return err
	}

	for _, r := range results {
		// Find all indirect referrers
		referrerNode := node.AddPath(r.ArtifactType, r.Digest)
		if opts.Verbose {
			for k, v := range r.Annotations {
				bytes, err := yaml.Marshal(map[string]string{k: v})
				if err != nil {
					return err
				}
				referrerNode.AddPath(strings.TrimSpace(string(bytes)))
			}
		}
		err := fetchAllReferrers(
			ctx, repo,
			ocispec.Descriptor{
				Digest:    r.Digest,
				Size:      r.Size,
				MediaType: r.MediaType,
			},
			artifactType, referrerNode, opts)
		if err != nil {
			return err
		}
	}
	return nil
}

func printDiscoveredReferrersTable(refs []ocispec.Descriptor, verbose bool) error {
	typeNameTitle := "Artifact Type"
	typeNameLength := len(typeNameTitle)
	for _, ref := range refs {
		if length := len(ref.ArtifactType); length > typeNameLength {
			typeNameLength = length
		}
	}

	print := func(key string, value interface{}) {
		fmt.Println(key, strings.Repeat(" ", typeNameLength-len(key)+1), value)
	}

	print(typeNameTitle, "Digest")
	for _, ref := range refs {
		print(ref.ArtifactType, ref.Digest)
		if verbose {
			if err := printJSON(ref); err != nil {
				return fmt.Errorf("error printing JSON: %w", err)
			}
		}
	}
	return nil
}

// printDiscoveredReferrersJSON prints referrer list in JSON equivalent to the
// image index: https://github.com/opencontainers/image-spec/blob/v1.1.0-rc2/image-index.md#image-index-property-descriptions
func printDiscoveredReferrersJSON(desc ocispec.Descriptor, refs []ocispec.Descriptor) error {
	output := ocispec.Index{
		Versioned: specs.Versioned{
			SchemaVersion: 2, // historical value. does not pertain to OCI or docker version
		},
		MediaType: ocispec.MediaTypeImageIndex,
		Manifests: refs,
	}

	return printJSON(output)
}

func printJSON(object interface{}) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	return encoder.Encode(object)
}
