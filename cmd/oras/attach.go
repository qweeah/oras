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

package main

import (
	"context"
	"errors"
	"fmt"
	"sync"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/spf13/cobra"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content"
	"oras.land/oras-go/v2/content/file"
	"oras.land/oras/cmd/oras/internal/display"
	oerrors "oras.land/oras/cmd/oras/internal/errors"
	"oras.land/oras/cmd/oras/internal/option"
	"oras.land/oras/internal/oci"
)

type attachOptions struct {
	option.Common
	option.Remote
	option.Packer

	targetRef    string
	artifactType string
	concurrency  int64
}

func attachCmd() *cobra.Command {
	var opts attachOptions
	cmd := &cobra.Command{
		Use:   "attach [flags] --artifact-type=<type> <name>{:<tag>|@<digest>} <file>[:<type>] [...]",
		Short: "[Preview] Attach files to an existing artifact",
		Long: `[Preview] Attach files to an existing artifact

** This command is in preview and under development. **

Example - Attach file 'hi.txt' with type 'doc/example' to manifest 'hello:test' in registry 'localhost:5000'
  oras attach --artifact-type doc/example localhost:5000/hello:test hi.txt

Example - Attach file 'hi.txt' and add annotations from file 'annotation.json'
  oras attach --artifact-type doc/example --annotation-file annotation.json localhost:5000/hello:latest hi.txt

Example - Attach an artifact with manifest annotations
  oras attach --artifact-type doc/example --annotation "key1=val1" --annotation "key2=val2" localhost:5000/hello:latest

Example - Attach file 'hi.txt' and add manifest annotations
  oras attach --artifact-type doc/example --annotation "key=val" localhost:5000/hello:latest hi.txt

Example - Attach file 'hi.txt' and export the pushed manifest to 'manifest.json'
  oras attach --artifact-type doc/example --export-manifest manifest.json localhost:5000/hello:latest hi.txt
`,
		Args: cobra.MinimumNArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return opts.ReadPassword()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.targetRef = args[0]
			opts.FileRefs = args[1:]
			return runAttach(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.artifactType, "artifact-type", "", "", "artifact type")
	cmd.Flags().Int64VarP(&opts.concurrency, "concurrency", "", 5, "concurrency level")
	cmd.MarkFlagRequired("artifact-type")
	option.ApplyFlags(&opts, cmd.Flags())
	return cmd
}

func runAttach(opts attachOptions) error {
	ctx, _ := opts.SetLoggerLevel()
	annotations, err := opts.LoadManifestAnnotations()
	if err != nil {
		return err
	}
	if len(opts.FileRefs) == 0 && len(annotations[option.AnnotationManifest]) == 0 {
		return errors.New("no blob or manifest annotation are provided")
	}

	// prepare manifest
	store := file.New("")
	defer store.Close()
	store.AllowPathTraversalOnWrite = opts.PathValidationDisabled

	dst, err := opts.NewRepository(opts.targetRef, opts.Common)
	if err != nil {
		return err
	}
	if dst.Reference.Reference == "" {
		return oerrors.NewErrInvalidReference(dst.Reference)
	}
	subject, err := dst.Resolve(ctx, dst.Reference.Reference)
	if err != nil {
		return err
	}
	blobs, err := loadFiles(ctx, store, annotations, opts.FileRefs, opts.Verbose)
	if err != nil {
		return err
	}
	packOpts := oras.PackOptions{
		Subject:             &subject,
		ManifestAnnotations: annotations[option.AnnotationManifest],
	}
	packFunc := oci.PackFunc(func(po oras.PackOptions) (ocispec.Descriptor, error) {
		return oras.Pack(ctx, store, opts.artifactType, blobs, po)
	})
	var committed sync.Map
	copyFunc := oci.CopyFunc(func(root ocispec.Descriptor) error {
		o := display.UploadOption(store, &committed, opts.concurrency, opts.Verbose, blobs)
		findSuccessors := o.FindSuccessors
		o.FindSuccessors = func(ctx context.Context, fetcher content.Fetcher, desc ocispec.Descriptor) ([]ocispec.Descriptor, error) {
			successors, err := findSuccessors(ctx, fetcher, desc)
			if err != nil {
				return nil, err
			}
			if !isEqualOCIDescriptor(desc, root) {
				return successors, nil
			}

			// skip subject to save one HEAD towards dst
			j := len(successors) - 1
			for i, s := range successors {
				if isEqualOCIDescriptor(s, subject) {
					// swap subject to end and slice it off
					successors[i] = successors[j]
					return o.FindSuccessors(ctx, fetcher, desc)
				}
			}
			return nil, fmt.Errorf("failed to find subject %v in the packed root %v", subject, root)
		}
		return oras.CopyGraph(ctx, store, dst, root, o)
	})

	// push
	root, err := oci.Upload(packOpts, packFunc, copyFunc, dst)
	if err != nil {
		return err
	}

	fmt.Println("Attached to", opts.targetRef)
	fmt.Println("Digest:", root.Digest)

	// Export manifest
	return opts.ExportManifest(ctx, store, root)
}

func isEqualOCIDescriptor(a, b ocispec.Descriptor) bool {
	return a.Size == b.Size && a.Digest == b.Digest && a.MediaType == b.MediaType
}
