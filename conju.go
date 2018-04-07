package conju

import (
	"html/template"

	"google.golang.org/appengine/log"
)

// TODO: Change this to the real link once we're live, or get it
// dynamically somehow.
const SiteLink = "http://localhost:8080"

func init() {
	AddSessionHandler("/reloadData", AskReloadData).Needs(AdminGetter)
	AddSessionHandler("/doReloadData", ReloadData).Needs(AdminGetter)
	AddSessionHandler("/clearData", ClearAllData).Needs(AdminGetter)

	AddSessionHandler("/listPeople", handleListPeople).Needs(EventGetter).Needs(AdminGetter)
	AddSessionHandler("/updatePersonForm", handleUpdatePersonForm).Needs(EventGetter).Needs(AdminGetter)
	AddSessionHandler("/saveUpdatePerson", handleSaveUpdatePerson).Needs(AdminGetter)
	AddSessionHandler("/invitations", handleInvitations).Needs(EventGetter).Needs(AdminGetter)
	AddSessionHandler("/copyInvitations", handleCopyInvitations).Needs(EventGetter).Needs(AdminGetter)
	AddSessionHandler("/addInvitation", handleAddInvitation).Needs(EventGetter).Needs(AdminGetter)
	AddSessionHandler("/viewInvitation", handleViewInvitationAdmin).Needs(EventGetter).Needs(AdminGetter)
	AddSessionHandler("/saveInvitation", handleSaveInvitation).Needs(EventGetter).Needs(LoginGetter)

	AddSessionHandler("/login", handleLogin("/rsvp")).Needs(EventGetter)
	AddSessionHandler(loginErrorPage, handleLoginError).Needs(EventGetter)
	AddSessionHandler("/rsvp", handleViewInvitationUser).Needs(LoginGetter)

	AddSessionHandler("/resendInvitation", handleResendInvitation).Needs(EventGetter)

	AddSessionHandler("/", handleIndex).Needs(EventGetter)
}

func handleIndex(wr WrappedRequest) {

	log.Infof(wr.Context, "wr.TemplateData: %v", wr.TemplateData)
	var tpl = template.Must(template.ParseFiles("templates/main.html", "templates/"+wr.Event.ShortName+"/index.html"))
	if err := tpl.ExecuteTemplate(wr.ResponseWriter, "index.html", wr.TemplateData); err != nil {
		log.Errorf(wr.Context, "%v", err)
	}
}
