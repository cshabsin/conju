package conju

import (
	"html/template"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

func Register(r *gin.Engine) {
	r.GET("/", handleIndex)

	adminGroup := r.Group("/", AdminMiddleware())
	adminGroup.GET("/admin", handleAdmin)
	adminGroup.GET("/events", handleEvents)
	adminGroup.POST("/createUpdateEvent", handleCreateUpdateEvent)
	adminGroup.GET("/invitations", handleInvitations)
	adminGroup.POST("/copyInvitations", handleCopyInvitations)
	adminGroup.POST("/addInvitation", handleAddInvitation)
	adminGroup.POST("/deleteInvitation", handleDeleteInvitation)
	adminGroup.GET("/viewInvitation", handleViewInvitationAdmin)

	r.GET("/login", handleLogin("/rsvp"))
	r.GET(loginErrorPage, handleLoginError)
	r.GET("/logout", handleLogout)
	r.POST("/resendInvitation", handleResendInvitation)
	r.GET(resentInvitationPage, handleResentInvitation)

	r.POST("/saveInvitation", handleSaveInvitation)

	r.GET("/poll", HandlePoll)

	// The rest of the routes will be migrated here.
}

func handleIndex(c *gin.Context) {
	wr, ok := c.MustGet("wrappedRequest").(*WrappedRequest)
	if !ok {
		log.Printf("could not get wrapped request from context")
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	eventName := "PSR2025"
	if wr.Event != nil {
		eventName = wr.Event.ShortName
	}
	var tpl = template.Must(template.ParseFiles("templates/main.html", "templates/"+eventName+"/index.html"))
	if err := tpl.ExecuteTemplate(c.Writer, "index.html", wr.TemplateData); err != nil {
		log.Println(err)
	}
}

func handleAdmin(c *gin.Context) {
	wr, ok := c.MustGet("wrappedRequest").(*WrappedRequest)
	if !ok {
		log.Printf("could not get wrapped request from context")
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	var tpl = template.Must(template.ParseFiles("templates/main.html", "templates/admin.html"))
	if err := tpl.ExecuteTemplate(c.Writer, "admin.html", wr.TemplateData); err != nil {
		log.Println(err)
	}
}

