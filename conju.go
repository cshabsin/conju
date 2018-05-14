package conju

import (
	"html/template"
	"math/rand"
	"time"

	"google.golang.org/appengine/log"
)

const SiteLink = "https://psr2018.shabsin.com"

func init() {

	AddSessionHandler("/reloadData", AskReloadData).Needs(AdminGetter)
	AddSessionHandler("/doReloadData", ReloadData).Needs(AdminGetter)
	//AddSessionHandler("/clearData", ClearAllData).Needs(AdminGetter)
	AddSessionHandler("/repairData", RepairData).Needs(AdminGetter)

	AddSessionHandler("/reloadHousingSetup", AskReloadHousingSetup).Needs(AdminGetter)
	AddSessionHandler("/doReloadHousingSetup", ReloadHousingSetup).Needs(AdminGetter)
	//	AddSessionHandler("/clearHousingSetup", ClearAllHousingSetup).Needs(AdminGetter)

	AddSessionHandler("/login", handleLogin("/rsvp"))
	AddSessionHandler(loginErrorPage, handleLoginError)
	AddSessionHandler("/logout", handleLogout)
	AddSessionHandler("/resendInvitation", handleResendInvitation)
	AddSessionHandler(resentInvitationPage, handleResentInvitation)

	AddSessionHandler("/admin", handleAdmin).Needs(PersonGetter).Needs(AdminGetter)

	AddSessionHandler("/listPeople", handleListPeople).Needs(PersonGetter).Needs(AdminGetter)
	AddSessionHandler("/updatePersonForm", handleUpdatePersonForm).Needs(PersonGetter).Needs(AdminGetter)
	AddSessionHandler("/saveUpdatePerson", handleSaveUpdatePerson).Needs(PersonGetter).Needs(AdminGetter)

	AddSessionHandler("/invitations", handleInvitations).Needs(PersonGetter).Needs(AdminGetter)
	AddSessionHandler("/copyInvitations", handleCopyInvitations).Needs(PersonGetter).Needs(AdminGetter)
	AddSessionHandler("/addInvitation", handleAddInvitation).Needs(PersonGetter).Needs(AdminGetter)
	AddSessionHandler("/viewInvitation", handleViewInvitationAdmin).Needs(PersonGetter).Needs(AdminGetter)
	AddSessionHandler("/saveInvitation", handleSaveInvitation).Needs(InvitationGetter)

	AddSessionHandler("/rsvp", handleViewInvitationUser).Needs(InvitationGetter)

	AddSessionHandler("/rsvpReport", handleRsvpReport).Needs(PersonGetter).Needs(AdminGetter)
	AddSessionHandler("/activitiesReport", handleActivitiesReport).Needs(PersonGetter).Needs(AdminGetter)
	AddSessionHandler("/roomingReport", handleRoomingReport).Needs(PersonGetter).Needs(AdminGetter)

	AddSessionHandler("/rooming", handleRoomingTool).Needs(PersonGetter).Needs(AdminGetter)
	AddSessionHandler("/saveRooming", handleSaveRooming).Needs(PersonGetter).Needs(AdminGetter)

	AddSessionHandler("/sendMail", handleSendMail).Needs(InvitationGetter).Needs(AdminGetter)
	AddSessionHandler("/doSendMail", handleDoSendMail).Needs(InvitationGetter).Needs(AdminGetter)

	AddSessionHandler("/", handleIndex).Needs(PersonGetter)
	//AddSessionHandler("/map", handleLoadMap).Needs(PersonGetter)

	rand.Seed(time.Now().UnixNano())

}

func handleIndex(wr WrappedRequest) {
	var tpl = template.Must(template.ParseFiles("templates/main.html", "templates/"+wr.Event.ShortName+"/index.html"))
	if err := tpl.ExecuteTemplate(wr.ResponseWriter, "index.html", wr.TemplateData); err != nil {
		log.Errorf(wr.Context, "%v", err)
	}
}

func handleLoadMap(wr WrappedRequest) {
	var tpl = template.Must(template.ParseFiles("templates/main.html", "templates/map.html"))
	if err := tpl.ExecuteTemplate(wr.ResponseWriter, "map.html", wr.TemplateData); err != nil {
		log.Errorf(wr.Context, "%v", err)
	}
}

func handleAdmin(wr WrappedRequest) {
	var tpl = template.Must(template.ParseFiles("templates/main.html", "templates/admin.html"))
	if err := tpl.ExecuteTemplate(wr.ResponseWriter, "admin.html", wr.TemplateData); err != nil {
		log.Errorf(wr.Context, "%v", err)
	}
}
