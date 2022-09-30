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

package command

import (
	"fmt"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "oras.land/oras/test/e2e/internal/utils"
)

const (
	preview_desc = "** This command is in preview and under development. **"
	example_desc = "\nExample - "
)

var _ = Describe("ORAS beginners", func() {
	Context("run manifest command", func() {
		runAndShowPreviewInHelp([]string{"manifest"})
		runAndShowPreviewInHelp([]string{"manifest", "fetch"}, preview_desc, example_desc)

		When("call sub-commands with aliases", func() {
			Success("manifest", "get", "--help").
				MatchKeyWords("[Preview] Fetch", preview_desc, example_desc).
				Exec("should succeed")
		})
		When("fetching manifest with no artifact reference provided", func() {
			Error("manifest", "fetch").MatchErrKeyWords("Error:").Exec("should fail")
		})
	})
})

func runAndShowPreviewInHelp(args []string, keywords ...string) {
	When(fmt.Sprintf("running %q command", strings.Join(args, " ")), func() {
		Success(append(args, "--help")...).
			MatchKeyWords(append(keywords, "[Preview] "+args[len(args)-1], "\nUsage:")...).
			Exec("should show preview and help doc")
	})
}
