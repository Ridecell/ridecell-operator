package actions

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/envy"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/markbates/goth/providers/github"
)

func init() {
	gothic.Store = App().SessionStore

	// Bail if github secrets aren't set
	gk, err := envy.MustGet("GITHUB_KEY")
	if err != nil {
		panic(err)
	}
	gs, err := envy.MustGet("GITHUB_SECRET")
	if err != nil {
		panic(err)
	}
	goth.UseProviders(
		github.New(gk, gs, fmt.Sprintf("%s%s", App().Host, "/auth/github/callback"), "read:org", "repo"),
	)
}

// AuthCallback is used to determine is user has permission to authenticate.
func AuthCallback(c buffalo.Context) error {
	user, err := gothic.CompleteUserAuth(c.Response(), c.Request())
	if err != nil {
		return err
	}

	c.Session().Set("current_user", user)
	err = c.Session().Save()
	if err != nil {
		return err
	}
	return c.Redirect(302, "/")
}

// Logout just drops the session, mostly just for testing purposes.
func Logout(c buffalo.Context) error {
	c.Session().Clear()
	err := c.Session().Save()
	if err != nil {
		return err
	}
	return c.Redirect(302, "/")
}

// Authorize is middleware to check if a user is allowed to do the thing.
func Authorize(next buffalo.Handler) buffalo.Handler {
	return func(c buffalo.Context) error {
		accessDenied := c.Error(403, errors.New("Access Denied"))

		cu := c.Session().Get("current_user")
		if cu == nil {
			return accessDenied
		}

		user, ok := cu.(goth.User)
		if !ok {
			return accessDenied
		}

		client := &http.Client{}

		// Check if user is a member of our org
		membershipCheckURL := fmt.Sprintf("https://api.github.com/orgs/Ridecell/members/%s", user.NickName)
		req, err := http.NewRequest("GET", membershipCheckURL, nil)
		if err != nil {
			return accessDenied
		}
		req.Header.Add("Authorization", fmt.Sprintf("token %s", user.AccessToken))

		resp, err := client.Do(req)
		if err != nil {
			return accessDenied
		}
		defer resp.Body.Close()

		// Check if user has access to a repo
		req1, err := http.NewRequest("GET", "https://api.github.com/repos/Ridecell/kubernetes-summon", nil)
		if err != nil {
			return accessDenied
		}
		req1.Header.Add("Authorization", fmt.Sprintf("token %s", user.AccessToken))

		resp1, err := client.Do(req1)
		if err != nil {
			return accessDenied
		}
		defer resp1.Body.Close()

		if resp.StatusCode != 204 {
			return accessDenied
		}
		if resp1.StatusCode != 200 {
			return accessDenied
		}

		return next(c)
	}
}

// SetCurrentUser is middleware to set current_user into request context.
func SetCurrentUserName(next buffalo.Handler) buffalo.Handler {
	return func(c buffalo.Context) error {
		cu := c.Session().Get("current_user")
		if cu == nil {
			return next(c)
		}
		user, ok := cu.(goth.User)
		if !ok {
			return next(c)
		}
		c.Set("current_username", user.Name)
		return next(c)
	}
}
