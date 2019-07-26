package gcr

import (
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/heroku/docker-registry-client/registry"
)

type parsedTag struct {
	Tag   string
	build int
}

type parsedTagList []parsedTag

func (a parsedTagList) Len() int           { return len(a) }
func (a parsedTagList) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a parsedTagList) Less(i, j int) bool { return a[i].build > a[j].build }

func GetLatestImageVersions() ([]string, error) {
	key := os.Getenv("GOOGLE_SERVICE_ACCOUNT_KEY")

	transport := registry.WrapTransport(http.DefaultTransport, "https://us.gcr.io", "_json_key", key)
	hub := &registry.Registry{
		URL: "https://us.gcr.io",
		Client: &http.Client{
			Transport: transport,
		},
		Logf: registry.Quiet,
	}

	tags, err := hub.Tags("ridecell-1/summon")
	if err != nil {
		return nil, err
	}

	var tagList []parsedTag
	for _, i := range tags {
		buildNumStr := strings.Split(i, "-")[0]
		buildNum, err := strconv.Atoi(buildNumStr)
		if err != nil {
			continue
		}
		tagList = append(tagList, parsedTag{build: buildNum, Tag: i})
	}

	sort.Sort(parsedTagList(tagList))

	var sortedTagList []string
	for _, i := range tagList[0:50] {
		sortedTagList = append(sortedTagList, i.Tag)
	}

	return sortedTagList, nil
}
