package conju

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"google.golang.org/appengine/v2/user"
)

func AdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		wr, ok := c.MustGet("wrappedRequest").(*WrappedRequest)
		if !ok {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		if wr.IsAdminUser() {
			c.Next()
			return
		}

		u := wr.User
		if u == nil {
			url, err := user.LoginURL(c.Request.Context(), c.Request.RequestURI)
			if err != nil {
				c.String(http.StatusInternalServerError, "Error generating login URL: %v", err)
				c.Abort()
				return
			}
			c.Header("Content-Type", "text/html; charset=utf-8")
			c.String(http.StatusForbidden, `This page requires administrator access. Please <a href="%s">Sign in</a>.`, url)
			c.Abort()
			return
		}

		logout_url, err := user.LogoutURL(c.Request.Context(), c.Request.RequestURI)
		if err != nil {
			c.String(http.StatusInternalServerError, "Error generating logout URL: %v", err)
			c.Abort()
			return
		}

		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(http.StatusForbidden, `This page requires administrator access.<br>User <code>&lt;%s&gt;</code> is not an authorized administrator.<p>Please <a href="%s">sign out</a> to try another account.`, u.Email, logout_url)
		c.Abort()
	}
}
