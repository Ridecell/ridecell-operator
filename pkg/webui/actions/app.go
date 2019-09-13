package actions

import (
	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/envy"
	"github.com/markbates/goth/gothic"
	"github.com/unrolled/secure"

	csrf "github.com/gobuffalo/mw-csrf"
	forcessl "github.com/gobuffalo/mw-forcessl"
	paramlogger "github.com/gobuffalo/mw-paramlogger"
)

// ENV is used to help switch settings based on where the
// application is being run. Default is "development".
var ENV = envy.Get("GO_ENV", "development")
var app *buffalo.App

// App is where all routes and middleware for buffalo
// should be defined. This is the nerve center of your
// application.
//
// Routing, middleware, groups, etc... are declared TOP -> DOWN.
// This means if you add a middleware to `app` *after* declaring a
// group, that group will NOT have that new middleware. The same
// is true of resource declarations as well.
//
// It also means that routes are checked in the order they are declared.
// `ServeFiles` is a CATCH-ALL route, so it should always be
// placed last in the route declarations, as it will prevent routes
// declared after it to never be called.
func App() *buffalo.App {
	host, err := envy.MustGet("HOST")
	if err != nil {
		panic(err)
	}

	if app == nil {
		app = buffalo.New(buffalo.Options{
			Env:         ENV,
			SessionName: "_webui_session",
			Addr:        "0.0.0.0:3000",
			Host:        host,
		})

		// Automatically redirect to SSL
		app.Use(forceSSL())

		// Override custom error handling in buffalo
		app.ErrorHandlers[403] = func(status int, err error, c buffalo.Context) error {
			res := c.Response()
			res.WriteHeader(403)
			_, nerr := res.Write([]byte("Access Denied"))
			if nerr != nil {
				return nerr
			}
			return nil
		}

		// reroute 404 -> 403 to prevent leaking structure
		app.ErrorHandlers[404] = func(status int, err error, c buffalo.Context) error {
			res := c.Response()
			res.WriteHeader(403)
			_, nerr := res.Write([]byte("Access Denied"))
			if nerr != nil {
				return nerr
			}
			return nil
		}

		// Log request parameters (filters apply).
		app.Use(paramlogger.ParameterLogger)

		// Protect against CSRF attacks. https://www.owasp.org/index.php/Cross-Site_Request_Forgery_(CSRF)
		// Remove to disable this.
		app.Use(csrf.New)

		// Setup and use translations:
		// app.Use(translations())

		app.GET("/", HomeHandler)
		app.Use(SetCurrentUserName)
		app.Use(Authorize)
		// Don't require auth for login page
		app.Middleware.Skip(Authorize, HomeHandler)

		auth := app.Group("/auth")
		gothHandler := buffalo.WrapHandlerFunc(gothic.BeginAuthHandler)
		auth.GET("/{provider}", gothHandler)
		auth.GET("/{provider}/callback", AuthCallback)
		// Don't require auth for things related to granting auth
		auth.Middleware.Skip(Authorize, gothHandler, AuthCallback)

		app.GET("/logout", Logout)
		app.Middleware.Skip(Authorize, Logout)

		app.POST("/pullrequest", CreatePR)

		statusGroup := app.Group("/status")
		statusGroup.GET("/", StatusBaseHandler)
		statusGroup.GET("/{instance}", StatusHandler)

		app.ServeFiles("/", assetsBox) // serve files from the public directory
	}

	return app
}

// forceSSL will return a middleware that will redirect an incoming request
// if it is not HTTPS. "http://example.com" => "https://example.com".
// This middleware does **not** enable SSL. for your application. To do that
// we recommend using a proxy: https://gobuffalo.io/en/docs/proxy
// for more information: https://github.com/unrolled/secure/
func forceSSL() buffalo.MiddlewareFunc {
	return forcessl.Middleware(secure.Options{
		SSLRedirect:     ENV == "production",
		SSLProxyHeaders: map[string]string{"X-Forwarded-Proto": "https"},
	})
}
