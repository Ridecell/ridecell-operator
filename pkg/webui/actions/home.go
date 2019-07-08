package actions

import "github.com/gobuffalo/buffalo"

// HomeHandler is a default handler to serve up
// a home page.
func HomeHandler(c buffalo.Context) error {
	if cu := c.Session().Get("current_user"); cu != nil {
		return c.Redirect(302, "/status")
	}
	return c.Render(200, r.HTML("index.html"))
}
