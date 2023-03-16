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

package utils

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"oras.land/oras-go/v2/registry"
)

// ORASPath points to the to-be-tested oras binary.
var ORASPath string

// Host points to the registry service where E2E specs will be run against.
var Host string

// FallbackHost points to the registry service where fallback E2E specs will be run against.
var FallbackHost string

// Path to generate the coverage report.
var CovDumpPath string

func init() {
	Host = os.Getenv(RegHostKey)
	if Host == "" {
		Host = "localhost:5000"
		fmt.Fprintf(os.Stderr, "cannot find host name in %s, using %s instead\n", RegHostKey, Host)
	}
	ref := registry.Reference{
		Registry: Host,
	}
	if err := ref.ValidateRegistry(); err != nil {
		panic(err)
	}

	FallbackHost = os.Getenv(FallbackRegHostKey)
	if FallbackHost == "" {
		FallbackHost = "localhost:6000"
		fmt.Fprintf(os.Stderr, "cannot find fallback host name in %s, using %s instead\n", FallbackRegHostKey, FallbackHost)
	}
	ref.Registry = FallbackHost
	if err := ref.ValidateRegistry(); err != nil {
		panic(err)
	}

	// setup test data
	pwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	// to simplify debugging via `go test`, TestDataRoot cannot be passed via ginkgo argument or env var
	TestDataRoot = filepath.Join(pwd, "..", "..", "testdata")
	if fi, err := os.Stat(TestDataRoot); err != nil || !fi.IsDir() {
		panic(fmt.Errorf("filed to find test data in %q", TestDataRoot))
	}
	BeforeSuite(func() {
		if covDumpRoot := os.Getenv("COVERAGE_DUMP_ROOT"); covDumpRoot != "" {
			if filepath.IsAbs(covDumpRoot) {
				CovDumpPath = covDumpRoot
			} else if workspacePath := os.Getenv("GITHUB_WORKSPACE"); workspacePath != "" {
				CovDumpPath = filepath.Join(workspacePath, "test/e2e", covDumpRoot)
			} else {
				// local debugging
				CovDumpPath = filepath.Join(pwd, "..", "..", covDumpRoot)
			}

			// confirm the existence of dump folder
			err := os.MkdirAll(CovDumpPath, 0700)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			fmt.Printf("Coverage file dump path: %q\n", CovDumpPath)
		}
		ORASPath = os.Getenv("ORAS_PATH")
		if filepath.IsAbs(ORASPath) {
			fmt.Printf("Testing based on pre-built binary locates in %q\n", ORASPath)
		} else if workspacePath := os.Getenv("GITHUB_WORKSPACE"); ORASPath != "" && workspacePath != "" {
			// add workspacePath as prefix, both path env should not be empty
			ORASPath = filepath.Join(workspacePath, ORASPath)
			ORASPath, err = filepath.Abs(ORASPath)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			fmt.Printf("Testing based on pre-built binary locates in %q\n", ORASPath)
		} else {
			// fallback to native build to facilitate local debugging
			buildArgs := []string{}
			if CovDumpPath != "" {
				buildArgs = append(buildArgs, "-coverpkg", "oras.land/oras")
			}
			ORASPath, err = gexec.Build("oras.land/oras/cmd/oras", buildArgs...)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			DeferCleanup(gexec.CleanupBuildArtifacts)
			fmt.Printf("Testing based on temp binary locates in %q\n", ORASPath)
		}

		// Login
		cmd := exec.Command(ORASPath, "login", Host, "-u", Username, "-p", Password)
		gomega.Expect(cmd.Run()).ShouldNot(gomega.HaveOccurred())
		cmd = exec.Command(ORASPath, "login", FallbackHost, "-u", Username, "-p", Password)
		gomega.Expect(cmd.Run()).ShouldNot(gomega.HaveOccurred())
	})
}
