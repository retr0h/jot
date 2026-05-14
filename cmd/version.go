// Copyright (c) 2026 John Dewey

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to
// deal in the Software without restriction, including without limitation the
// rights to use, copy, modify, merge, publish, distribute, sublicense, and/or
// sell copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
// FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER
// DEALINGS IN THE SOFTWARE.

package cmd

import (
	"fmt"

	goversion "github.com/caarlos0/go-version"
	"github.com/spf13/cobra"

	"github.com/retr0h/jot/internal/version"
)

func buildVersion(ver, commit, date, builtBy string) goversion.Info {
	return goversion.GetVersionInfo(
		goversion.WithAppDetails(
			"jot",
			"terminal notes + todos with linked tasks.\n",
			"https://github.com/retr0h/jot",
		),
		func(i *goversion.Info) {
			if commit != "" {
				i.GitCommit = commit
			}
			if date != "" {
				i.BuildDate = date
			}
			if ver != "" {
				i.GitVersion = ver
			}
			if builtBy != "" {
				i.BuiltBy = builtBy
			}
		},
	)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Display the version of jot",
	RunE: func(_ *cobra.Command, _ []string) error {
		v := buildVersion(version.Version, version.Commit, version.Date, version.BuiltBy)
		jsonOut, err := v.JSONString()
		if err != nil {
			return fmt.Errorf("failed to render version JSON: %w", err)
		}
		fmt.Println(jsonOut)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
