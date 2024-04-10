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
	"errors"
	"fmt"
	"os"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/spf13/cobra"

	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/registry"
	"oras.land/oras/cmd/oras/internal/argument"
	"oras.land/oras/cmd/oras/internal/display"
	"oras.land/oras/cmd/oras/internal/display/metadata/template"
	oerrors "oras.land/oras/cmd/oras/internal/errors"
	"oras.land/oras/cmd/oras/internal/option"
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
			return option.Parse(cmd, &opts)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDiscover(cmd, &opts)
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
	return oerrors.Command(cmd, &opts.Target)
}

func runDiscover(cmd *cobra.Command, opts *discoverOptions) error {
	ctx, logger := opts.WithContext(cmd.Context())
	repo, err := opts.NewReadonlyTarget(ctx, opts.Common, logger)
	if err != nil {
		return err
	}
	if err := opts.EnsureReferenceNotEmpty(cmd, true); err != nil {
		return err
	}

	// discover artifacts
	resolveOpts := oras.DefaultResolveOptions
	resolveOpts.TargetPlatform = opts.Platform.Platform
	desc, err := oras.Resolve(ctx, repo, opts.Reference, resolveOpts)
	if err != nil {
		return err
	}

	var referrers = func(d ocispec.Descriptor) ([]ocispec.Descriptor, error) {
		return registry.Referrers(ctx, repo, d, opts.artifactType)
	}
	if opts.Template != "" {
		handler := display.NewDiscoverHandler(cmd.OutOrStdout(), opts.Template, opts.Path, opts.RawReference, desc, referrers, opts.Verbose)
		return handler.OnDiscovered()
	}

	// deprecated --output
	fmt.Fprintf(os.Stderr, "[DEPRECATED] --output is deprecated, try `--format %s` instead\n", opts.outputType)
	handler := display.NewDiscoverHandler(cmd.OutOrStdout(), opts.outputType, opts.Path, opts.RawReference, desc, referrers, opts.Verbose)
	if _, ok := handler.(*template.DiscoverHandler); ok {
		return errors.New("output type can only be tree, table or json")
	}
	return handler.OnDiscovered()
}
