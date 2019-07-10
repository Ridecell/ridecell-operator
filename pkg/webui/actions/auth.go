package actions

import (
	"fmt"
	"io/ioutil"
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
		return c.Error(401, err)
	}

	membershipCheckURL := fmt.Sprintf("https://api.github.com/orgs/Ridecell/members/%s", user.NickName)
	req, err := http.NewRequest("GET", membershipCheckURL, nil)
	if err != nil {
		return err
	}
	req.Header.Add("Authorization", fmt.Sprintf("token %s", user.AccessToken))

	req1, err := http.NewRequest("GET", "https://api.github.com/repos/Ridecell/kubernetes-summon", nil)
	if err != nil {
		return err
	}
	req1.Header.Add("Authorization", fmt.Sprintf("token %s", user.AccessToken))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	resp1, err := client.Do(req1)
	if err != nil {
		return err
	}
	defer resp1.Body.Close()
	body, err := ioutil.ReadAll(resp1.Body)
	if err != nil {
		return err
	}
	fmt.Printf("%#v\n", string(body))

	// User is not a member of org, don't save the session.']
	if resp.StatusCode != 204 {
		c.Flash().Add("danger", "Access Denied.")
		return c.Redirect(302, "/")
	}

	fmt.Printf("%#v\n", user)
	fmt.Println(user.AccessToken)

	c.Flash().Add("success", "Successfully logged in.")
	c.Session().Set("current_user", user)
	err = c.Session().Save()
	if err != nil {
		return err
	}
	// Do something with the user, maybe register them/sign them in
	return c.Redirect(302, "/")
}

// Logout just drops the session, mostly just for testing purposes.
func Logout(c buffalo.Context) error {
	c.Flash().Add("success", "Successfully logged out.")
	c.Session().Clear()
	err := c.Session().Save()
	if err != nil {
		return err
	}
	return c.Redirect(302, "/")
}

// Authorize is middleware to check if a user is authenticated.
func Authorize(next buffalo.Handler) buffalo.Handler {
	return func(c buffalo.Context) error {
		cu := c.Session().Get("current_user")
		if cu == nil {
			c.Flash().Add("danger", "Access Denied.")
			return c.Redirect(302, "/")
		}
		return next(c)
	}
}

// SetCurrentUser is middleware to set current_user into request context.
func SetCurrentUser(next buffalo.Handler) buffalo.Handler {
	return func(c buffalo.Context) error {
		cu := c.Session().Get("current_user")
		if cu == nil {
			return next(c)
		}
		c.Set("current_user", cu)
		return next(c)
	}
}
