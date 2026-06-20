package version

import (
	"testing"

	"github.com/Bugs5382/gitlab-release-drafter/internal/model"
)

func TestParse(t *testing.T) {
	cases := map[string]Semver{
		"v1.2.3": {1, 2, 3},
		"1.2.3":  {1, 2, 3},
		"v0.0.1": {0, 0, 1},
	}
	for in, want := range cases {
		got, err := Parse(in)
		if err != nil {
			t.Fatalf("Parse(%q): %v", in, err)
		}
		if got != want {
			t.Fatalf("Parse(%q) = %v, want %v", in, got, want)
		}
	}
	if _, err := Parse("nope"); err == nil {
		t.Fatal("Parse(nope) should error")
	}
}

func TestBaseHighestMatching(t *testing.T) {
	tags := []string{"v1.0.0", "v1.4.2", "v1.4.10", "release-9", "v0.9.0"}
	got := Base(tags, "v{version}", "0.1.0")
	if got.String() != "1.4.10" {
		t.Fatalf("Base = %s, want 1.4.10", got)
	}
}

func TestBaseFallsBackToInitial(t *testing.T) {
	if got := Base(nil, "v{version}", "0.1.0"); got.String() != "0.1.0" {
		t.Fatalf("Base = %s, want 0.1.0", got)
	}
	if got := Base([]string{"weird"}, "v{version}", "2.0.0"); got.String() != "2.0.0" {
		t.Fatalf("Base = %s, want 2.0.0", got)
	}
}

func TestResolveHighestIncrement(t *testing.T) {
	inc := map[string][]string{
		"major": {"breaking"},
		"minor": {"feature"},
		"patch": {"fix"},
	}
	mrs := []model.MergeRequest{
		{Labels: []string{"feature"}},
		{Labels: []string{"breaking"}},
		{Labels: []string{"fix"}},
	}
	if got := Resolve(mrs, inc, "patch"); got != Major {
		t.Fatalf("Resolve = %v, want Major", got)
	}
}

func TestResolveDefaultWhenNoMatch(t *testing.T) {
	inc := map[string][]string{"major": {"breaking"}, "minor": {"feature"}, "patch": {"fix"}}
	mrs := []model.MergeRequest{{Labels: []string{"docs"}}}
	if got := Resolve(mrs, inc, "minor"); got != Minor {
		t.Fatalf("Resolve = %v, want Minor (default)", got)
	}
}

func TestBump(t *testing.T) {
	cases := []struct {
		base Semver
		inc  Increment
		want Semver
	}{
		{Semver{1, 4, 7}, Major, Semver{2, 0, 0}},
		{Semver{1, 4, 7}, Minor, Semver{1, 5, 0}},
		{Semver{1, 4, 7}, Patch, Semver{1, 4, 8}},
	}
	for _, c := range cases {
		if got := Bump(c.base, c.inc); got != c.want {
			t.Fatalf("Bump(%v,%v) = %v, want %v", c.base, c.inc, got, c.want)
		}
	}
}

func TestRender(t *testing.T) {
	v := Semver{2, 0, 1}
	if got := RenderTag("v{version}", v); got != "v2.0.1" {
		t.Fatalf("RenderTag = %s", got)
	}
	if got := RenderTag("{major}.{minor}", v); got != "2.0" {
		t.Fatalf("RenderTag major.minor = %s", got)
	}
}
