package app

import (
	"os"
	"strconv"
	"strings"

	"github.com/Bugs5382/gitlab-release-drafter/internal/autolabel"
	"github.com/Bugs5382/gitlab-release-drafter/internal/config"
	"github.com/Bugs5382/gitlab-release-drafter/internal/model"
)

func autolabelMatch(mr model.MergeRequest, rules []config.Autolabel) []string {
	return autolabel.Match(mr, rules)
}

// changelogItem renders a single MR into a changelog bullet item.
func changelogItem(tmpl string, mr model.MergeRequest) string {
	r := strings.NewReplacer(
		"{title}", mr.Title,
		"{iid}", strconv.Itoa(mr.IID),
		"{author}", mr.Author.Username,
		"{url}", mr.WebURL,
		"{labels}", strings.Join(mr.Labels, ", "),
	)
	return r.Replace(tmpl)
}

func osReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}
