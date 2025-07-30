package conju

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"cloud.google.com/go/datastore"
	"github.com/cshabsin/conju/conju/dsclient"
	"github.com/cshabsin/conju/conju/mailer"
	"github.com/cshabsin/conju/conju/mailer/forwardemail"
	"github.com/cshabsin/conju/conju/secretmanager"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/sessions"
	"google.golang.org/appengine/v2"
	"google.golang.org/appengine/v2/user"
)

type GinMiddleware struct {
	DatastoreClient     *datastore.Client
	SecretmanagerClient *secretmanager.Client
	MailClient          *forwardemail.Client
	cookieStore         *sessions.CookieStore
}

func NewGinMiddleware(ds *datastore.Client, sm *secretmanager.Client, mail *forwardemail.Client) *GinMiddleware {
	return &GinMiddleware{
		DatastoreClient:     ds,
		SecretmanagerClient: sm,
		MailClient:          mail,
	}
}

func (m *GinMiddleware) initializeCookieStore(ctx context.Context) error {
	if m.cookieStore != nil {
		return nil
	}
	log.Printf("initializing cookie store")
	smClient, err := secretmanager.FromContext(ctx)
	if err != nil {
		return fmt.Errorf("Error getting secret manager: %w", err)
	}
	secrets, err := smClient.GetVersionsWithinGracePeriod(ctx, "gorilla_cookie_auth_key", 0)
	if err != nil {
		return fmt.Errorf("Error getting gorilla cookie auth key: %w", err)
	}
	var secretArgs [][]byte
	for _, secret := range secrets {
		secretArgs = append(secretArgs, secret)
		secretArgs = append(secretArgs, nil)
	}
	m.cookieStore = sessions.NewCookieStore(secretArgs...)
	return nil
}

func (m *GinMiddleware) SessionMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := dsclient.WrapContext(appengine.NewContext(c.Request), m.DatastoreClient)
		ctx = secretmanager.WrapContext(ctx, m.SecretmanagerClient)
		ctx = mailer.WrapContext(ctx, m.MailClient)

		if err := m.initializeCookieStore(ctx); err != nil {
			log.Printf("Could not initialize secret manager in sessionizer: %v", err)
			m.cookieStore = sessions.NewCookieStore([]byte("devmode_key_crsdms"))
		}

		sess, err := m.cookieStore.Get(c.Request, "conju")
		if err != nil {
			log.Printf("Could not get session from store: %v", err)
			http.Error(c.Writer, err.Error(), http.StatusInternalServerError)
			c.Abort()
			return
		}
		u := user.Current(ctx)
		wr := &WrappedRequest{
			Context: c,
			Session: sess,
			User:    u,
			TemplateData: map[string]any{
				"User": u,
			},
		}
		if u != nil {
			logoutUrl, err := user.LogoutURL(ctx, wr.Request.URL.RequestURI())
			if err == nil {
				wr.TemplateData["LogoutLink"] = logoutUrl
			}
		}
		wr.TemplateData["IsAdminUser"] = wr.IsAdminUser()
		wr.TemplateData["DevMode"] = wr.IsAdminUser() && len(wr.Request.URL.Query()["devmode"]) > 0

		c.Set("wrappedRequest", wr)
		c.Next()
	}
}