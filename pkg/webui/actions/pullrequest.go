package actions

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/gobuffalo/buffalo"
	"github.com/google/go-github/github"
	"github.com/markbates/goth"
	"golang.org/x/oauth2"
)

// TODO: Clean this mess

const owner = "Ridecell"
const repoName = "kubernetes-summon"

// CreatePR is a handler to serve the PR creation page.
func CreatePR(c buffalo.Context) error {
	dockerTag := c.Param("docker-tag")
	instanceName := c.Param("instance-name")
	newCommitMessage := fmt.Sprintf("Update %s version to %s", instanceName, dockerTag)
	newBranchName := fmt.Sprintf("update-%s-to-%s", instanceName, dockerTag)

	ctx := context.Background()

	user, err := getGothUserFromSession(c)
	if err != nil {
		return err
	}

	tc := oauth2.NewClient(ctx, oauth2.StaticTokenSource(&oauth2.Token{AccessToken: user.AccessToken}))
	client := github.NewClient(tc)

	branch, _, err := client.Repositories.GetBranch(ctx, owner, repoName, "master")
	if err != nil {
		return err
	}

	masterTree, _, err := client.Git.GetTree(ctx, owner, repoName, branch.Commit.GetSHA(), true)
	if err != nil {
		return err
	}

	masterCommit, _, err := client.Git.GetCommit(ctx, owner, repoName, branch.Commit.GetSHA())
	if err != nil {
		return err
	}

	// Split the instance name to get our target filename
	splits := strings.Split(instanceName, "-")

	// Locate our target file
	var targetFile github.TreeEntry
	targetFileIndex := -1
	for i, v := range masterTree.Entries {
		if strings.Contains(v.GetPath(), fmt.Sprintf("%s.yml", splits[0])) {
			targetFile = v
			targetFileIndex = i
			break
		}
	}

	// If we couldn't find the file exit out
	if targetFileIndex == -1 {
		return errors.New("failed to find specified instance in repository")
	}

	// Fetch raw text blob for file instead of dealing with base64
	blobBytes, _, err := client.Git.GetBlobRaw(ctx, owner, repoName, targetFile.GetSHA())
	if err != nil {
		return err
	}

	// This is a bad way of doing this but it works i guess
	newData := regexp.MustCompile(`(version: )(\S*)`).ReplaceAllString(string(blobBytes), fmt.Sprintf("${1}%s", dockerTag))
	newEncodedData := base64.StdEncoding.EncodeToString([]byte(newData))
	if err != nil {
		return err
	}

	// Create a new blob for our file
	newBlob, _, err := client.Git.CreateBlob(ctx, owner, repoName, &github.Blob{
		Content:  getStringPointer(newEncodedData),
		Encoding: getStringPointer("base64"),
	})
	if err != nil {
		return err
	}

	// Copy tree from master branch over to new slice and replace our targeted file
	newTreeEntries := make([]github.TreeEntry, len(masterTree.Entries))
	copy(newTreeEntries, masterTree.Entries)
	newTreeEntries[targetFileIndex] = github.TreeEntry{
		Path: targetFile.Path,
		Mode: getStringPointer("100644"),
		Type: getStringPointer("blob"),
		SHA:  newBlob.SHA,
	}

	// Create our new tree with modified contents
	newTree, _, err := client.Git.CreateTree(ctx, owner, repoName, masterTree.GetSHA(), newTreeEntries)
	if err != nil {
		return err
	}

	// Create a new commit using our modified tree and the master branch head sha as parent
	newCommit, _, err := client.Git.CreateCommit(ctx, owner, repoName, &github.Commit{
		Message: getStringPointer(newCommitMessage),
		Tree:    newTree,
		Parents: []github.Commit{*masterCommit},
	})
	if err != nil {
		return err
	}

	// Create a new branch referencing our new commit
	_, _, err = client.Git.CreateRef(ctx, owner, repoName, &github.Reference{
		Ref:    getStringPointer(fmt.Sprintf("refs/heads/%s", newBranchName)),
		Object: &github.GitObject{SHA: newCommit.SHA},
	})
	if err != nil {
		return err
	}

	// Create the pull request
	newPR, _, err := client.PullRequests.Create(ctx, owner, repoName, &github.NewPullRequest{
		Title: getStringPointer(newCommitMessage),
		Head:  getStringPointer(newBranchName),
		Base:  getStringPointer("master"),
		Body:  getStringPointer(""),
	})
	if err != nil {
		return err
	}

	// Redirect user to the newly created pull request
	return c.Redirect(302, newPR.GetHTMLURL())
}

func getGothUserFromSession(c buffalo.Context) (*goth.User, error) {
	cu := c.Session().Get("current_user")
	if cu == nil {
		return nil, errors.New("unable to fetch authed user from session")
	}

	user, ok := cu.(goth.User)
	if !ok {
		return nil, errors.New("unable to convert current_user to goth user type")
	}
	return &user, nil
}

// Helper func cause string pointers are annoying
func getStringPointer(input string) *string {
	return &input
}
