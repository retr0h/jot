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

// Package cmd contains the jot cobra command tree.
package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/lmittmann/tint"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/term"

	"github.com/retr0h/jot/internal/cli"
	"github.com/retr0h/jot/internal/config"
)

// logger is the package-level slog logger, populated from initLogger
// after cobra parses persistent flags. CLI subcommands log through it
// directly.
var (
	appConfig  config.Config
	logger     = slog.New(slog.NewTextHandler(os.Stderr, nil))
	jsonOutput bool
)

var rootCmd = &cobra.Command{
	Use:   "jot",
	Short: "Terminal notes + todos with linked tasks",
}

// Execute runs the root command; invoked by main. SilenceUsage drops
// the help-text dump on runtime failures where it's just noise. Cobra
// already prints "Error: <err>" on its own.
func Execute() {
	rootCmd.SilenceUsage = true

	// Wrap cobra's default help to print the themed banner above it.
	// SetHelpFunc fires for `jot --help` and for the bare-command
	// fallback alike, so the banner shows in both paths without
	// duplicating itself.
	defaultHelp := rootCmd.HelpFunc()
	rootCmd.SetHelpFunc(func(c *cobra.Command, args []string) {
		if c == rootCmd {
			out := c.OutOrStdout()
			_, _ = fmt.Fprintln(out)
			_, _ = fmt.Fprint(out, cli.Banner(out))
			_, _ = fmt.Fprintln(out)
		}
		defaultHelp(c, args)
	})

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// ConfigDir returns the resolved jot config directory.
func ConfigDir() string {
	return appConfig.Config
}

// NotesDir returns the directory where jot stores note files.
func NotesDir() string {
	return appConfig.NotesDir
}

// SSHKeys returns the configured SSH key paths for kvlt identity resolution.
func SSHKeys() []string {
	return appConfig.SSHKeys
}

func init() {
	cobra.OnInitialize(initConfig, initLogger)

	home, _ := os.UserHomeDir()
	rootCmd.PersistentFlags().String(
		"config", filepath.Join(home, ".config", "jot"),
		"config directory (default ~/.config/jot)",
	)
	rootCmd.PersistentFlags().BoolP("debug", "d", false, "enable debug logging")
	rootCmd.PersistentFlags().BoolVarP(&jsonOutput, "json", "j", false, "emit logs as JSON")
	rootCmd.PersistentFlags().String("notes-dir", "", "notes directory")
	rootCmd.PersistentFlags().StringSlice("ssh-key", nil, "SSH key path(s) for kvlt (repeatable)")

	_ = viper.BindPFlag("config", rootCmd.PersistentFlags().Lookup("config"))
	_ = viper.BindPFlag("notes_dir", rootCmd.PersistentFlags().Lookup("notes-dir"))
	_ = viper.BindPFlag("ssh_keys", rootCmd.PersistentFlags().Lookup("ssh-key"))
	_ = viper.BindPFlag("debug", rootCmd.PersistentFlags().Lookup("debug"))
}

// initConfig wires viper — env-var overrides take effect through a
// JOT_… prefix with dots replaced by underscores, so e.g.
// JOT_NOTES_DIR overrides the notes_dir default.
func initConfig() {
	viper.SetEnvPrefix("jot")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// Load jot.yaml from the config directory when it exists.
	viper.SetConfigName("jot")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(viper.GetString("config"))
	_ = viper.ReadInConfig()

	// Defaults — all overridable via config file, env vars, or flags.
	viper.SetDefault("notes_dir", "")

	_ = viper.Unmarshal(&appConfig)

	if appConfig.NotesDir == "" {
		appConfig.NotesDir = filepath.Join(appConfig.Config, "notes")
	}
}

// initLogger swaps the package-level logger to a tint handler with
// color when stderr is a TTY, plain text otherwise. --json swaps in
// the slog JSON handler — for log aggregators that prefer structured
// input. Level follows --debug.
func initLogger() {
	level := slog.LevelInfo
	if viper.GetBool("debug") {
		level = slog.LevelDebug
	}

	var handler slog.Handler
	if jsonOutput {
		handler = slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: level})
	} else {
		handler = tint.NewHandler(os.Stderr, &tint.Options{
			Level:      level,
			TimeFormat: time.Kitchen,
			NoColor:    !term.IsTerminal(int(os.Stderr.Fd())),
		})
	}

	logger = slog.New(handler)
	slog.SetDefault(logger)
}
