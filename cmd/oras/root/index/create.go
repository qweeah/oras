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

package index

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sync"

	"github.com/opencontainers/go-digest"
	"github.com/opencontainers/image-spec/specs-go"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content"
	"oras.land/oras-go/v2/errdef"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras/cmd/oras/internal/display"
	"oras.land/oras/cmd/oras/internal/display/track"
	"oras.land/oras/cmd/oras/internal/option"
	"oras.land/oras/internal/docker"
	"oras.land/oras/internal/graph"
)

type createOptions struct {
	option.Common
	option.Descriptor
	option.Pretty
	option.Target

	recursive   bool
	concurrency int
	subject     string
	mediaType   string
	srcs        []option.Target
}

func createCmd() *cobra.Command {
	var opts createOptions
	cmd := &cobra.Command{
		Use:   "create [flags] <name>[:<tag>|@<digest>] [...]",
		Short: "create a index to from provided manifests",
		Long: `create a index to a registry or an OCI image layout

Example - create a index to repository 'localhost:5000/hello' and tag with 'v1':
  oras index create localhost:5000/hello:v1 \
     localhost:5000/hello@sha256:xxxx \
     localhost:5000/hello@sha256:xxxx
`,
		Args: cobra.MinimumNArgs(2),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			opts.RawReference = args[0]
			// TODO: args are not enough, requires platform and thus
			// another option with dedicated parsing
			opts.srcs = make([]option.Target, len(args)-1)
			for i, a := range args[1:] {
				m := option.Target{RawReference: a}
				if err := m.Parse(); err != nil {
					return err
				}
				opts.srcs[i] = m
			}
			return option.Parse(&opts)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return createManifest(cmd.Context(), opts)
		},
	}

	opts.EnableDistributionSpecFlag()
	option.ApplyFlags(&opts, cmd.Flags())
	cmd.Flags().BoolVarP(&opts.recursive, "recursive", "r", false, "[Preview] recursively copy the artifact and its referrer artifacts")
	cmd.Flags().StringVarP(&opts.mediaType, "media-type", "", ocispec.MediaTypeImageIndex, "media type of index")
	cmd.Flags().IntVarP(&opts.concurrency, "concurrency", "", 3, "concurrency level")
	cmd.Flags().StringVarP(&opts.subject, "subject", "", "", "subject artifact of the index")
	return cmd
}

func createManifest(ctx context.Context, opts createOptions) error {
	ctx, logger := opts.WithContext(ctx)
	// todo: annotataion
	// annotations, err := opts.LoadManifestAnnotations()
	// if err != nil {
	// 	return err
	// }
	// Prepare dest target
	dst, err := opts.NewTarget(opts.Common, logger)
	if err != nil {
		return err
	}
	manifests, err := doCopy(ctx, dst, opts, logger)
	if err != nil {
		return err
	}
	if err := doPack(ctx, dst, manifests, opts); err != nil {
		return err
	}
	display.Print("created", opts.AnnotatedReference())

	return nil
}

func doCopy(ctx context.Context, dst oras.GraphTarget, destOpts createOptions, logger logrus.FieldLogger) ([]ocispec.Descriptor, error) {
	// Prepare copy options
	committed := &sync.Map{}
	baseCopyOptions := oras.DefaultExtendedCopyOptions
	baseCopyOptions.Concurrency = destOpts.concurrency
	baseCopyOptions.FindPredecessors = func(ctx context.Context, src content.ReadOnlyGraphStorage, desc ocispec.Descriptor) ([]ocispec.Descriptor, error) {
		return graph.Referrers(ctx, src, desc, "")
	}
	const (
		promptExists  = "Exists "
		promptCopying = "Copying"
		promptCopied  = "Copied "
		promptSkipped = "Skipped"
		promptMounted = "Mouned "
	)
	var onMounted func(context.Context, ocispec.Descriptor) error
	if destOpts.TTY == nil {
		// none TTY output
		baseCopyOptions.OnCopySkipped = func(ctx context.Context, desc ocispec.Descriptor) error {
			committed.Store(desc.Digest.String(), desc.Annotations[ocispec.AnnotationTitle])
			return display.PrintStatus(desc, promptExists, destOpts.Verbose)
		}
		baseCopyOptions.PreCopy = func(ctx context.Context, desc ocispec.Descriptor) error {
			return display.PrintStatus(desc, promptCopying, destOpts.Verbose)
		}
		baseCopyOptions.PostCopy = func(ctx context.Context, desc ocispec.Descriptor) error {
			committed.Store(desc.Digest.String(), desc.Annotations[ocispec.AnnotationTitle])
			if err := display.PrintSuccessorStatus(ctx, desc, dst, committed, display.StatusPrinter(promptSkipped, destOpts.Verbose)); err != nil {
				return err
			}
			return display.PrintStatus(desc, promptCopied, destOpts.Verbose)
		}
		onMounted = func(ctx context.Context, desc ocispec.Descriptor) error {
			committed.Store(desc.Digest.String(), desc.Annotations[ocispec.AnnotationTitle])
			return display.PrintStatus(desc, promptMounted, destOpts.Verbose)
		}
	} else {
		// TTY output
		tracked, err := track.NewTarget(dst, promptCopying, promptCopied, destOpts.TTY)
		if err != nil {
			return nil, err
		}
		defer tracked.Close()
		dst = tracked
		baseCopyOptions.OnCopySkipped = func(ctx context.Context, desc ocispec.Descriptor) error {
			committed.Store(desc.Digest.String(), desc.Annotations[ocispec.AnnotationTitle])
			return tracked.Prompt(desc, promptExists)
		}
		baseCopyOptions.PostCopy = func(ctx context.Context, desc ocispec.Descriptor) error {
			committed.Store(desc.Digest.String(), desc.Annotations[ocispec.AnnotationTitle])
			return display.PrintSuccessorStatus(ctx, desc, tracked, committed, func(desc ocispec.Descriptor) error {
				return tracked.Prompt(desc, promptSkipped)
			})
		}
		onMounted = func(ctx context.Context, desc ocispec.Descriptor) error {
			// todo: no deduplication + no repo name
			committed.Store(desc.Digest.String(), desc.Annotations[ocispec.AnnotationTitle])
			return tracked.Prompt(desc, promptMounted)
		}
	}

	// copy all manifests
	rOpts := oras.DefaultResolveOptions
	var copied []ocispec.Descriptor
	dstAsRemote, dstIsRemote := dst.(*remote.Repository)
	for _, srcOpts := range destOpts.srcs {
		var err error
		// prepare src target
		src, err := srcOpts.NewReadonlyTarget(ctx, destOpts.Common, logger)
		if err != nil {
			return copied, err
		}
		if err := srcOpts.EnsureReferenceNotEmpty(); err != nil {
			// todo: make me fail fast
			return nil, err
		}
		srcAsRemote, srcIsRemote := src.(*remote.Repository)

		copyOptions := baseCopyOptions
		if srcIsRemote && dstIsRemote && srcAsRemote.Reference.Registry == dstAsRemote.Reference.Registry && srcAsRemote.Reference.Repository != dstAsRemote.Reference.Repository {
			copyOptions.WithMount(srcAsRemote.Reference.Repository, dstAsRemote, onMounted)
		}
		var desc ocispec.Descriptor
		if destOpts.recursive {
			desc, err = oras.Resolve(ctx, src, srcOpts.Reference, rOpts)
			if err != nil {
				return copied, fmt.Errorf("failed to resolve %s: %w", srcOpts.Reference, err)
			}
			err = recursiveCopy(ctx, src, dst, desc.Digest.String(), desc, copyOptions)
		} else {
			desc, err = oras.Resolve(ctx, src, srcOpts.Reference, rOpts)
			if err != nil {
				return copied, fmt.Errorf("failed to resolve %s: %w", srcOpts.Reference, err)
			}
			err = oras.CopyGraph(ctx, src, dst, desc, copyOptions.CopyGraphOptions)
		}
		if err != nil {
			return copied, err
		}
		copied = append(copied, desc)
	}

	return copied, nil
}

func doPack(ctx context.Context, t oras.Target, manifests []ocispec.Descriptor, opts createOptions) error {
	// todo: oras-go needs PackIndex
	index := ocispec.Index{
		Versioned: specs.Versioned{
			SchemaVersion: 2,
		},
		MediaType: opts.mediaType,
		Manifests: manifests,
		// todo: annotations
	}
	content, _ := json.Marshal(index)
	reader := bytes.NewReader(content)
	desc := ocispec.Descriptor{
		Digest:    digest.FromBytes(content),
		MediaType: opts.mediaType,
		Size:      int64(len(content)),
	}

	const (
		promptUploading = "Uploading"
		promptUploaded  = "Uploaded "
	)
	if opts.TTY == nil {
		// none TTY output
		if err := display.PrintStatus(desc, promptUploading, opts.Verbose); err != nil {
			return err
		}
		if err := t.Push(ctx, desc, reader); err != nil {
			w := errors.Unwrap(err)
			if w != errdef.ErrAlreadyExists {
				return err
			}
		}
		return display.PrintStatus(desc, promptUploaded, opts.Verbose)
	}

	// TTY output
	trackedReader, err := track.NewReader(reader, desc, promptUploading, promptUploaded, opts.TTY)
	if err != nil {
		return err
	}
	defer trackedReader.StopManager()
	trackedReader.Start()
	if err := t.Push(ctx, desc, trackedReader); err != nil {
		w := errors.Unwrap(err)
		if w != errdef.ErrAlreadyExists {
			return err
		}
	}
	trackedReader.Done()
	return nil
}

// todo: duplicated to cp's
// recursiveCopy copies an artifact and its referrers from one target to another.
// If the artifact is a manifest list or index, referrers of its manifests are copied as well.
func recursiveCopy(ctx context.Context, src oras.ReadOnlyGraphTarget, dst oras.Target, dstRef string, root ocispec.Descriptor, opts oras.ExtendedCopyOptions) error {
	if root.MediaType == ocispec.MediaTypeImageIndex || root.MediaType == docker.MediaTypeManifestList {
		fetched, err := content.FetchAll(ctx, src, root)
		if err != nil {
			return err
		}
		var index ocispec.Index
		if err = json.Unmarshal(fetched, &index); err != nil {
			return nil
		}

		referrers, err := graph.FindPredecessors(ctx, src, index.Manifests, opts)
		if err != nil {
			return err
		}
		referrers = slices.DeleteFunc(referrers, func(desc ocispec.Descriptor) bool {
			return content.Equal(desc, root)
		})

		findPredecessor := opts.FindPredecessors
		opts.FindPredecessors = func(ctx context.Context, src content.ReadOnlyGraphStorage, desc ocispec.Descriptor) ([]ocispec.Descriptor, error) {
			descs, err := findPredecessor(ctx, src, desc)
			if err != nil {
				return nil, err
			}
			if content.Equal(desc, root) {
				// make sure referrers of child manifests are copied by pointing them to root
				descs = append(descs, referrers...)
			}
			return descs, nil
		}
	}

	var err error
	if dstRef == "" || dstRef == root.Digest.String() {
		err = oras.ExtendedCopyGraph(ctx, src, dst, root, opts.ExtendedCopyGraphOptions)
	} else {
		_, err = oras.ExtendedCopy(ctx, src, root.Digest.String(), dst, dstRef, opts)
	}
	return err
}
