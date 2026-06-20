package config

import (
	"encoding/base64"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config is the full GLRD_CONFIG schema (spec §8).
type Config struct {
	Version struct {
		TagTemplate      string              `yaml:"tag-template"`
		NameTemplate     string              `yaml:"name-template"`
		Initial          string              `yaml:"initial"`
		DefaultIncrement string              `yaml:"default-increment"`
		Increments       map[string][]string `yaml:"increments"`
	} `yaml:"version"`

	Categories         []Category `yaml:"categories"`
	UncategorizedTitle string     `yaml:"uncategorized-title"`

	LabelsMode    string   `yaml:"labels-mode"`
	ExcludeLabels []string `yaml:"exclude-labels"`

	Template          string `yaml:"template"`
	ChangeTemplate    string `yaml:"change-template"`
	NoChangesTemplate string `yaml:"no-changes-template"`
	CategoryTemplate  string `yaml:"category-template"`
	Sort              string `yaml:"sort"`
	SortDirection     string `yaml:"sort-direction"`

	Contributors Contributors `yaml:"contributors"`
	Autolabeler  []Autolabel  `yaml:"autolabeler"`
	Changelog    Changelog    `yaml:"changelog"`
}

type Category struct {
	Title  string   `yaml:"title"`
	Labels []string `yaml:"labels"`
}

type Contributors struct {
	Exclude   []string `yaml:"exclude"`
	Template  string   `yaml:"template"`
	Separator string   `yaml:"separator"`
}

type Autolabel struct {
	Label  string   `yaml:"label"`
	Branch []string `yaml:"branch"`
	Title  []string `yaml:"title"`
	Files  []string `yaml:"files"`
}

type Changelog struct {
	File           string            `yaml:"file"`
	KeepAChangelog bool              `yaml:"keep-a-changelog"`
	SectionMap     map[string]string `yaml:"section-map"`
}

// Load parses config from a GLRD_CONFIG value. The value is YAML, optionally
// base64-encoded: if it does not parse as YAML, a base64 decode is attempted.
func Load(raw string) (*Config, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, fmt.Errorf("config: GLRD_CONFIG is empty")
	}

	cfg, err := parse(raw)
	if err != nil {
		decoded, decErr := base64.StdEncoding.DecodeString(strings.TrimSpace(raw))
		if decErr != nil {
			return nil, fmt.Errorf("config: not valid YAML and not base64: %w", err)
		}
		cfg, err = parse(string(decoded))
		if err != nil {
			return nil, fmt.Errorf("config: base64-decoded value is not valid YAML: %w", err)
		}
	}

	cfg.applyDefaults()
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func parse(s string) (*Config, error) {
	var cfg Config
	dec := yaml.NewDecoder(strings.NewReader(s))
	dec.KnownFields(false)
	if err := dec.Decode(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (c *Config) applyDefaults() {
	if c.Version.TagTemplate == "" {
		c.Version.TagTemplate = "v{version}"
	}
	if c.Version.NameTemplate == "" {
		c.Version.NameTemplate = "v{version}"
	}
	if c.Version.Initial == "" {
		c.Version.Initial = "0.1.0"
	}
	if c.Version.DefaultIncrement == "" {
		c.Version.DefaultIncrement = "patch"
	}
	if c.LabelsMode == "" {
		c.LabelsMode = "any"
	}
	if c.Template == "" {
		c.Template = "{changes}"
	}
	if c.ChangeTemplate == "" {
		c.ChangeTemplate = "- {title} (!{iid}) by @{author}"
	}
	if c.NoChangesTemplate == "" {
		c.NoChangesTemplate = "No notable changes."
	}
	if c.CategoryTemplate == "" {
		c.CategoryTemplate = "### {title}"
	}
	if c.Sort == "" {
		c.Sort = "merged_at"
	}
	if c.SortDirection == "" {
		c.SortDirection = "desc"
	}
	if c.Contributors.Template == "" {
		c.Contributors.Template = "@{username}"
	}
	if c.Contributors.Separator == "" {
		c.Contributors.Separator = ", "
	}
	if c.Changelog.File == "" {
		c.Changelog.File = "CHANGELOG.md"
	}
}

func (c *Config) validate() error {
	switch c.LabelsMode {
	case "any", "all":
	default:
		return fmt.Errorf("config: labels-mode must be 'any' or 'all', got %q", c.LabelsMode)
	}
	switch c.Version.DefaultIncrement {
	case "major", "minor", "patch":
	default:
		return fmt.Errorf("config: version.default-increment must be major|minor|patch, got %q", c.Version.DefaultIncrement)
	}
	switch c.SortDirection {
	case "asc", "desc":
	default:
		return fmt.Errorf("config: sort-direction must be 'asc' or 'desc', got %q", c.SortDirection)
	}
	if c.Sort != "merged_at" && c.Sort != "title" {
		return fmt.Errorf("config: sort must be 'merged_at' or 'title', got %q", c.Sort)
	}
	if len(c.Categories) == 0 && strings.TrimSpace(c.UncategorizedTitle) == "" {
		return fmt.Errorf("config: at least one category or an uncategorized-title is required")
	}
	return nil
}
