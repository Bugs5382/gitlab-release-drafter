package main

import (
	"os"
	"strconv"
	"time"

	"github.com/spf13/cobra"

	"github.com/Bugs5382/gitlab-release-drafter/internal/app"
	"github.com/Bugs5382/gitlab-release-drafter/internal/config"
	"github.com/Bugs5382/gitlab-release-drafter/internal/gitlab"
)

// shared flag values bound by the root command.
var (
	flagMilestone string
	flagOutDir    string
	flagConfigVar string
	flagAutoTag   bool
)

const longDescription = `gitlab-release-drafter is a GitLab-native release drafter.

It resolves the next semantic version from a milestone's merged-MR labels,
renders release notes, maintains a Keep a Changelog CHANGELOG.md, and creates
GitLab Releases — all driven by GitLab CI.

Configuration comes from the GLRD_CONFIG environment variable (raw or base64
YAML). Authentication comes from GLRD_TOKEN (api scope).`

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "gitlab-release-drafter",
		Short:         "GitLab-native release drafter",
		Long:          longDescription,
		SilenceUsage:  true,
		SilenceErrors: false,
	}

	pf := root.PersistentFlags()
	pf.StringVar(&flagMilestone, "milestone", os.Getenv("GLRD_MILESTONE"), "milestone title or IID override")
	pf.StringVar(&flagOutDir, "out-dir", ".", "directory for artifacts and CHANGELOG")
	pf.StringVar(&flagConfigVar, "config-var", "GLRD_CONFIG", "env var holding the YAML config")

	root.AddCommand(
		newPreviewCmd(),
		newLabelCmd(),
		newReleaseCmd(),
		newChangelogCmd(),
	)
	return root
}

func newPreviewCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "preview",
		Short: "Resolve version and render notes for the relevant milestone (read-only)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			d, err := buildDeps()
			if err != nil {
				return err
			}
			return app.Preview(cmd.Context(), d)
		},
	}
}

func newLabelCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "label",
		Short: "Apply autolabeler rules to the current pipeline's MR(s)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			d, err := buildDeps()
			if err != nil {
				return err
			}
			return app.Label(cmd.Context(), d)
		},
	}
}

func newReleaseCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "release",
		Short: "Resolve, tag (optional), create the Release, and update CHANGELOG",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			d, err := buildDeps()
			if err != nil {
				return err
			}
			return app.Release(cmd.Context(), d)
		},
	}
	cmd.Flags().BoolVar(&flagAutoTag, "auto-tag", envBool("GLRD_AUTO_TAG"), "create the resolved tag during release")
	return cmd
}

func newChangelogCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "changelog",
		Short: "Update CHANGELOG.md without cutting a release",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			d, err := buildDeps()
			if err != nil {
				return err
			}
			return app.Changelog(cmd.Context(), d)
		},
	}
}

func buildDeps() (*app.Deps, error) {
	cfg, err := config.Load(os.Getenv(flagConfigVar))
	if err != nil {
		return nil, err
	}

	client := gitlab.NewHTTPClient(gitlab.Options{
		APIBaseURL: os.Getenv("CI_API_V4_URL"),
		ProjectID:  os.Getenv("CI_PROJECT_ID"),
		Token:      os.Getenv("GLRD_TOKEN"),
		MRIID:      envInt("CI_MERGE_REQUEST_IID"),
	})

	return &app.Deps{
		Client: client,
		Config: cfg,
		Stdout: os.Stdout,
		Now:    time.Now().UTC(),
		Env: app.Env{
			Milestone:     flagMilestone,
			CommitBranch:  os.Getenv("CI_COMMIT_BRANCH"),
			DefaultBranch: os.Getenv("CI_DEFAULT_BRANCH"),
			CommitTag:     os.Getenv("CI_COMMIT_TAG"),
			ProjectURL:    os.Getenv("CI_PROJECT_URL"),
			AutoTag:       flagAutoTag,
			OutDir:        flagOutDir,
		},
	}, nil
}

func envBool(key string) bool {
	b, _ := strconv.ParseBool(os.Getenv(key))
	return b
}

func envInt(key string) int {
	n, _ := strconv.Atoi(os.Getenv(key))
	return n
}
