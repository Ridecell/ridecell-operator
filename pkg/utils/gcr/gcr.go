package gcr

import (
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/heroku/docker-registry-client/registry"
	"github.com/pkg/errors"
)

const CacheExpiry time.Duration = time.Minute * 5

// Global variable used for cache purposes.
var LastCacheUpdate time.Time
var CachedTags []string

func SanitizeBranchName(branch string) (string, error) {
	// Since the circleci build number probably won't go over 7 digits, truncate branch name to 48 chars to
	// match against docker image tag. Preceeding 16 chars left for [circlecibuild#]-[7 digit commit hash]-
	if len(branch) >= 48 {
		branch = branch[0:48]
	}

	// If last character of string is non-alphanumeric, replace with 'x'.
	reg, err := regexp.Compile("[^a-zA-Z0-9]$")
	if err != nil {
		return "", errors.Wrap(err, "regex [^a-zA-Z0-9]$ in GetLatestImageOfBranch()")
	}
	sanitized_branch_tag := reg.ReplaceAllString(branch, "x")

	// Replace non-alphanumeric with 'x', since docker image tags are sanitized this way.
	reg, err = regexp.Compile("[^a-zA-Z0-9_.-]")
	if err != nil {
		return "", errors.Wrap(err, "regex [^a-zA-Z0-9_.-] in GetLatestImageOfBranch()")
	}
	sanitized_branch_tag = reg.ReplaceAllString(sanitized_branch_tag, "-")
	return sanitized_branch_tag, nil
}

func GetLatestImageOfBranch(branchTag string) (string, error) {
	var latestImage string
	latestBuild := 0
	elapsed := time.Since(LastCacheUpdate)

	// Fetch tags if cache expired.
	if elapsed >= CacheExpiry {
		// Setup hub connection
		var key = os.Getenv("GOOGLE_SERVICE_ACCOUNT_KEY")
		var registry_url = os.Getenv("LOCAL_REGISTRY_URL")
		// If we don't have a test registry, use the real one.
		if registry_url == "" {
			registry_url = "https://us.gcr.io"
		}

		var transport = registry.WrapTransport(http.DefaultTransport, registry_url, "_json_key", key)
		var summonHub = &registry.Registry{
			URL: registry_url,
			Client: &http.Client{
				Transport: transport,
			},
			Logf: registry.Quiet,
		}

		tags, err := summonHub.Tags("ridecell-1/summon")
		if err != nil {
			return "", errors.Wrapf(err, "Could not retrieve tags from registry: ")
		}
		CachedTags = tags
		LastCacheUpdate = time.Now()
	}

	for _, image := range CachedTags {

		// Append $ so we do not match beyond the end of branchTag. This prevents
		// situations where we have similar branch names like "fix-for-ticket1" and "fix-for-ticket2"
		match, err := regexp.Match(regexp.QuoteMeta(branchTag)+"$", []byte(image))
		if err != nil {
			return "", errors.Wrapf(err, "regexp.Match(%s, []byte(%s)) in GetLatestImageOfBranch()", branchTag, image)
		}

		if match {
			// Expects docker image to follow format <circleci buildnum>-<git hash>-<branchname>
			buildNumStr := strings.Split(image, "-")[0]
			buildNum, err := strconv.Atoi(buildNumStr)
			if err != nil {
				glog.Infof("Failed to convert %s into an integer: %s", buildNumStr, err)
				continue
			}
			// Check for largest buildNumStr instead of running a sort, since we're doing O(n) anyway.
			if buildNum > latestBuild {
				latestBuild = buildNum
				latestImage = image
			}
		}
	}
	if glog.V(5) {
		glog.Infof("[autodeploy] Latest image for %s is %s", branchTag, latestImage)
	}
	return latestImage, nil
}
