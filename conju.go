package conju

import (
	"html/template"

	"google.golang.org/appengine/log"
)

const SiteLink = "https://psr2018.shabsin.com"

func init() {

	AddSessionHandler("/reloadData", AskReloadData).Needs(AdminGetter)
	AddSessionHandler("/doReloadData", ReloadData).Needs(AdminGetter)
	AddSessionHandler("/clearData", ClearAllData).Needs(AdminGetter)

	AddSessionHandler("/listPeople", handleListPeople).Needs(PersonGetter).Needs(AdminGetter)
	AddSessionHandler("/updatePersonForm", handleUpdatePersonForm).Needs(PersonGetter).Needs(AdminGetter)
	AddSessionHandler("/saveUpdatePerson", handleSaveUpdatePerson).Needs(PersonGetter).Needs(AdminGetter)
	AddSessionHandler("/invitations", handleInvitations).Needs(PersonGetter).Needs(AdminGetter)
	AddSessionHandler("/copyInvitations", handleCopyInvitations).Needs(PersonGetter).Needs(AdminGetter)
	AddSessionHandler("/addInvitation", handleAddInvitation).Needs(PersonGetter).Needs(AdminGetter)
	AddSessionHandler("/viewInvitation", handleViewInvitationAdmin).Needs(PersonGetter).Needs(AdminGetter)
	AddSessionHandler("/saveInvitation", handleSaveInvitation).Needs(InvitationGetter)

	AddSessionHandler("/login", handleLogin("/rsvp"))
	AddSessionHandler(loginErrorPage, handleLoginError)
	AddSessionHandler("/logout", handleLogout)
	AddSessionHandler("/rsvp", handleViewInvitationUser).Needs(InvitationGetter)

	AddSessionHandler("/sendMail", handleSendMail).Needs(InvitationGetter).Needs(AdminGetter)
	AddSessionHandler("/doSendMail", handleDoSendMail).Needs(InvitationGetter).Needs(AdminGetter)

	AddSessionHandler("/resendInvitation", handleResendInvitation)
	AddSessionHandler(resentInvitationPage, handleResentInvitation)

	AddSessionHandler("/", handleIndex).Needs(PersonGetter)
}

func handleIndex(wr WrappedRequest) {

	var tpl = template.Must(template.ParseFiles("templates/main.html", "templates/"+wr.Event.ShortName+"/index.html"))
	if err := tpl.ExecuteTemplate(wr.ResponseWriter, "index.html", wr.TemplateData); err != nil {
		log.Errorf(wr.Context, "%v", err)
	}
}
