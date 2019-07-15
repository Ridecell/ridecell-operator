package actions

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/Ridecell/ridecell-operator/pkg/webui/kubernetes"
	"github.com/gobuffalo/buffalo"
	"github.com/heroku/docker-registry-client/registry"
	"github.com/markbates/goth"
	"gopkg.in/src-d/go-billy.v4"
	"gopkg.in/src-d/go-billy.v4/memfs"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"gopkg.in/src-d/go-git.v4/plumbing/transport/http"
	"gopkg.in/src-d/go-git.v4/storage/memory"
)

type dockerconfig struct {
	Auths []map[string]string `json:"auths"`
}

type parsedTag struct {
	tag, sha, branch string
	build            int
}

type byBuild []parsedTag

func (a byBuild) Len() int           { return len(a) }
func (a byBuild) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byBuild) Less(i, j int) bool { return a[i].build > a[j].build }

// PRSetupHandler serves page for setting up a deployment pull request
func PRSetupHandler(c buffalo.Context) error {
	instanceName := c.Param("instance")
	instance, err := kubernetes.GetSummonObject(instanceName)
	if err != nil {
		return err
	}

	// Get our pull secret
	fi, err := os.Open("dockerconfig/.dockerconfigjson")
	if err != nil {
		return err
	}

	configBytes, err := ioutil.ReadAll(fi)
	if err != nil {
		return err
	}

	config := &dockerconfig{}
	err = json.Unmarshal(configBytes, config)
	if err != nil {
		return err
	}

	base64EncodedJSONKey, ok := config.Auths[0]["auth"]
	if !ok {
		return errors.New("Unable to find auth key")
	}

	jsonKey, err := base64.DecodeString(string(base64EncodedJSONKey))
	if err != nil {
		return errors.New("Unable to decode json key")
	}

	transport := registry.WrapTransport(http.DefaultTransport, "https://us.gcr.io", "_json_key", jsonKey)
	hub := &registry.Registry{
		URL: "https://us.gcr.io",
		Client: &http.Client{
			Transport: transport,
		},
		Logf: registry.Quiet,
	}

	// Get all tags for the summon image.
	tags, err := hub.Tags("ridecell-1/summon")
	if err != nil {
		return err
	}

	// Do some quick parsing.
	var parsedTags []parsedTag
	for _, tag := range tags {
		parts := regexp.MustCompile(`^(\d+)-([0-9a-f]+)-(.*)$`).FindStringSubmatch(tag)
		if parts == nil {
			// Not sure what that is.
			continue
		}
		build, err := strconv.Atoi(parts[1])
		if err != nil {
			return err
		}

		byBranch := map[string]parsedTag{}
		for _, parsed := range parsedTags {
			existing, ok := byBranch[parsed.branch]
			if !ok || parsed.build > existing.build {
				byBranch[parsed.branch] = parsed
			}
		}
		branchRegexp := regexp.MustCompile(`^(master$|release)`)
		branches := make([]string, 0, len(byBranch))
		for b := range byBranch {
			if branchRegexp.MatchString(b) {
				branches = append(branches, b)
			}
		}
		sort.Strings(branches)
		for _, b := range branches {
			parsed := byBranch[b]
			fmt.Printf("%s: %s\n", parsed.branch, parsed.tag)
		}
		parsedTags = append(parsedTags, parsedTag{tag: tag, build: build, sha: parts[2], branch: parts[3]})
	}

	c.Set("instance", instance)
	return c.Render(200, r.HTML("pullrequest/setup.html"))
}

// PRHandler is a handler for setting up a PR.
func PRHandler(c buffalo.Context) error {
	instanceName := c.Param("instance-name")

	// Convert stored user in session to usable goth object
	cu, ok := c.Session().Get("current_user").(goth.User)
	if !ok {
		return errors.New("could get convert goth user")
	}
	token := cu.AccessToken

	fs := memfs.New()
	rep, err := git.Clone(memory.NewStorage(), fs, &git.CloneOptions{
		URL: "https://github.com/Ridecell/kubernetes-summon",
		// github uses basic auth with tokens
		Auth: &http.BasicAuth{
			Username: "Username",
			Password: token,
		},
	})
	if err != nil {
		return err
	}

	wt, err := rep.Worktree()
	if err != nil {
		return err
	}

	headRef, err := rep.Head()
	if err != nil {
		return err
	}

	// Build a branch name
	branchName := fmt.Sprintf("%s/%s-update-version-%s", cu.NickName, instanceName, c.Param("Version"))
	// Get a new plumbing reference for it
	branchRefName := plumbing.NewBranchReferenceName(fmt.Sprintf("refs/heads/%s", branchName))
	// Build a new branch hash
	ref := plumbing.NewHashReference(branchRefName, headRef.Hash())

	// Store that branch hash
	err = rep.Storer.SetReference(ref)
	if err != nil {
		return err
	}

	// Checkout our new branch hash
	err = wt.Checkout(&git.CheckoutOptions{
		Hash: ref.Hash(),
	})
	if err != nil {
		return err
	}

	fpath, err := findManifest(instanceName, fs)

	f, err := fs.OpenFile(fpath, 0x2, 0777)
	if err != nil {
		return err
	}

	contents, err := ioutil.ReadAll(f)
	if err != nil {
		return err
	}

	stringContents := string(contents)

	oldString := regexp.MustCompile(`version: (.*)`).FindStringSubmatch(stringContents)
	if oldString == nil {
		return errors.New("unable to find version in file contents")
	}

	newString := strings.ReplaceAll(stringContents, oldString[1], c.Param("Version"))

	// Wipe the file clean before rewrite
	err = f.Truncate(0)
	if err != nil {
		return err
	}

	// Seek back to first byte before rewrite
	_, err = f.Seek(0, 0)
	if err != nil {
		return err
	}

	_, err = f.Write([]byte(newString))
	if err != nil {
		return err
	}

	defer f.Close()

	// Add file to git worktree
	wt.Add(fpath)

	commit, err := wt.Commit(fmt.Sprintf("Update %s version to %s", instanceName, c.Param("Version")), &git.CommitOptions{
		Author: &object.Signature{
			Name:  cu.Name,
			Email: cu.Email,
			When:  time.Now(),
		},
	})
	if err != nil {
		return err
	}

	obj, err := rep.CommitObject(commit)
	if err != nil {
		return err
	}

	fmt.Println(obj)

	//err = rep.Push(&git.PushOptions{
	//	RemoteName: "origin",
	//	RefSpecs:   []config.RefSpec{},
	//	Auth: &http.BasicAuth{
	//		Username: "Username",
	//		Password: token,
	//	},
	//})
	//if err != nil {
	//	return err
	//}

	return c.Render(200, r.String(newString))
}

func findManifest(instance string, fs billy.Filesystem) (string, error) {
	name, namespace := splitInstanceString(instance)

	dir, err := fs.ReadDir(".")
	if err != nil {
		return "", err
	}
	// hacky walk dir since billy.filesystem seems to only have basics
	for _, i := range dir {
		match := regexp.MustCompile(fmt.Sprintf(`^.*-%s`, namespace)).MatchString(i.Name())
		if i.IsDir() && match {
			deepDir, err := fs.ReadDir(i.Name())
			if err != nil {
				return "", err
			}
			for _, j := range deepDir {
				match := regexp.MustCompile(fmt.Sprintf(`^%s.yml`, name)).MatchString(j.Name())
				if match {
					return fmt.Sprintf("%s/%s", i.Name(), j.Name()), nil
				}
			}
		}
	}
	return "", errors.New("unable to find target file")
}

func splitInstanceString(instance string) (string, string) {
	splits := strings.Split(instance, "-")
	return splits[0], splits[1]
}
