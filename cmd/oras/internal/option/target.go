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

package option

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/pflag"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/oci"
	"oras.land/oras-go/v2/registry"
	"oras.land/oras/cmd/oras/internal/fileref"
)

const (
	TargetTypeRemote    = "registry"
	TargetTypeOCILayout = "oci-layout"
)

// Unary target option struct.
type Target struct {
	Remote
	RawReference string
	Type         string
	Reference    string //contains tag or digest

	isOCI bool
}

// ApplyFlags applies flags to a command flag set for unary target
func (opts *Target) ApplyFlags(fs *pflag.FlagSet) {
	opts.applyFlagsWithPrefix(fs, "", "")
	opts.Remote.ApplyFlags(fs)
}

// AnnotatedReference returns full printable reference.
func (opts *Target) AnnotatedReference() string {
	return fmt.Sprintf("[%s] %s", opts.Type, opts.RawReference)
}

// applyFlagsWithPrefix applies flags to a command flag set with a prefix string.
// The full target flag should be provided as:
//
//	--target type={oci-layout|registry}.
//
// Since only OCI image layout is supported, a short boolean flag
// `--[{from|to}-oci-layout]` is used to simplify UX.
//   - If the flag is set, it equals to `--target type=oci-layout`;
//   - If not, it equals to `--target type=registry`
func (opts *Target) applyFlagsWithPrefix(fs *pflag.FlagSet, prefix, description string) {
	var (
		flagPrefix string
		noteSuffix string
	)
	if prefix != "" {
		flagPrefix = prefix + "-"
		noteSuffix = description + " "
	}
	fs.BoolVarP(&opts.isOCI, flagPrefix+"TargetTypeOCILayout", "", false, "Set "+noteSuffix+"target as an OCI image layout.")
}

// ApplyFlagsWithPrefix applies flags to a command flag set with a prefix string.
// Commonly used for non-unary remote targets.
func (opts *Target) ApplyFlagsWithPrefix(fs *pflag.FlagSet, prefix, description string) {
	opts.applyFlagsWithPrefix(fs, prefix, description)
	opts.Remote.ApplyFlagsWithPrefix(fs, prefix, description)
}

// Parse gets target options from user input.
func (opts *Target) Parse() error {
	switch {
	case opts.isOCI:
		opts.Type = TargetTypeOCILayout
	default:
		opts.Type = TargetTypeRemote
	}
	return nil
}

// parseOCILayoutReference parses the raw in format of <path>[:<tag>|@<digest>]
func parseOCILayoutReference(raw string) (string, string, error) {
	var path, ref string
	var err error
	if idx := strings.LastIndex(raw, "@"); idx != -1 {
		// `digest` found
		path = raw[:idx]
		ref = raw[idx+1:]
	} else {
		// find `tag`
		path, ref, err = fileref.Parse(raw, "")
		if err != nil {
			return "", "", err
		}
	}
	return path, ref, nil
}

// NewTarget generates a new target based on opts.
func (opts *Target) NewTarget(common Common) (graphTarget oras.GraphTarget, err error) {
	switch opts.Type {
	case TargetTypeOCILayout:
		var path string
		path, opts.Reference, err = parseOCILayoutReference(opts.RawReference)
		if err != nil {
			return nil, err
		}
		graphTarget, err = oci.New(path)
		return
	case TargetTypeRemote:
		repo, err := opts.NewRepository(opts.RawReference, common)
		if err != nil {
			return nil, err
		}
		opts.Reference = repo.Reference.Reference
		return repo, nil
	}
	return nil, fmt.Errorf("unknown target type: %q", opts.Type)
}

// Read-only graph target with tag lister.
type ReadOnlyGraphTagFinderTarget interface {
	oras.ReadOnlyGraphTarget
	registry.TagLister
}

// NewReadonlyTargets generates a new read only target based on opts.
func (opts *Target) NewReadonlyTarget(ctx context.Context, common Common) (ReadOnlyGraphTagFinderTarget, error) {
	switch opts.Type {
	case TargetTypeOCILayout:
		var path string
		var err error
		path, opts.Reference, err = parseOCILayoutReference(opts.RawReference)
		if err != nil {
			return nil, err
		}
		info, err := os.Stat(path)
		if err != nil {
			return nil, err
		}
		if info.IsDir() {
			return oci.NewFromFS(ctx, os.DirFS(path))
		}
		return oci.NewFromTar(ctx, path)
	case TargetTypeRemote:
		repo, err := opts.NewRepository(opts.RawReference, common)
		if err != nil {
			return nil, err
		}
		opts.Reference = repo.Reference.Reference
		return repo, nil
	}
	return nil, fmt.Errorf("unknown target type: %q", opts.Type)
}

// Binary target option struct.
type BinaryTarget struct {
	From Target
	To   Target
}

// ApplyFlagsWithPrefix applies flags to a command flag set with a prefix string.
// Commonly used for non-unary remote targets.
func (opts *BinaryTarget) ApplyFlags(fs *pflag.FlagSet) {
	opts.From.ApplyFlagsWithPrefix(fs, "from", "source")
	opts.To.ApplyFlagsWithPrefix(fs, "to", "destination")
}

// Parse parses user-provided flags and arguments into option struct.
func (opts *BinaryTarget) Parse() error {
	if err := opts.From.Parse(); err != nil {
		return err
	}
	return opts.To.Parse()
}
