package conju

import (
	"context"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"time"

	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
)

// SiteLink holds the current URL to base absolute links on.
const SiteLink = "https://psr2021.shabsin.com"

func Register() {
	http.HandleFunc("/_ah/start", func(w http.ResponseWriter, r *http.Request) {
		datastore.EnableKeyConversion(appengine.NewContext(r))
	})

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
	AddSessionHandler("/receivePay", handleReceivePay).Needs(PersonGetter).Needs(AdminGetter)
	AddSessionHandler("/doReceivePay", handleDoReceivePay).Needs(PersonGetter).Needs(AdminGetter)
	AddSessionHandler("/copyInvitations", handleCopyInvitations).Needs(PersonGetter).Needs(AdminGetter)
	AddSessionHandler("/addInvitation", handleAddInvitation).Needs(PersonGetter).Needs(AdminGetter)
	AddSessionHandler("/deleteInvitation", handleDeleteInvitation).Needs(PersonGetter).Needs(AdminGetter)
	AddSessionHandler("/viewInvitation", handleViewInvitationAdmin).Needs(PersonGetter).Needs(AdminGetter)
	AddSessionHandler("/saveInvitation", handleSaveInvitation).Needs(InvitationGetter)

	AddSessionHandler("/events", handleEvents).Needs(PersonGetter).Needs(AdminGetter)
	AddSessionHandler("/createUpdateEvent", handleCreateUpdateEvent).Needs(PersonGetter).Needs(AdminGetter)

	AddSessionHandler("/rsvp", handleViewInvitationUser).Needs(InvitationGetter)

	AddSessionHandler("/rsvpReport", handleRsvpReport).Needs(PersonGetter).Needs(AdminGetter)
	AddSessionHandler("/activitiesReport", handleActivitiesReport).Needs(PersonGetter).Needs(AdminGetter)
	AddSessionHandler("/roomingReport", handleRoomingReport).Needs(PersonGetter).Needs(AdminGetter)
	AddSessionHandler("/handleSaveReservations", handleSaveReservations).Needs(PersonGetter).Needs(AdminGetter)
	AddSessionHandler("/foodReport", handleFoodReport).Needs(PersonGetter).Needs(AdminGetter)
	AddSessionHandler("/ridesReport", handleRidesReport).Needs(PersonGetter).Needs(AdminGetter)

	AddSessionHandler("/rooming", handleRoomingTool).Needs(PersonGetter).Needs(AdminGetter)
	AddSessionHandler("/saveRooming", handleSaveRooming).Needs(PersonGetter).Needs(AdminGetter)

	AddSessionHandler("/sendMail", handleSendMail).Needs(InvitationGetter).Needs(AdminGetter)
	AddSessionHandler("/doSendMail", handleDoSendMail).Needs(InvitationGetter).Needs(AdminGetter)

	AddSessionHandler("/testRoomingMail", handleTestSendRoomingEmail).Needs(AdminGetter)
	AddSessionHandler("/sendRoomingMail", handleAskSendRoomingEmail).Needs(AdminGetter)
	AddSessionHandler("/doSendTestRoomingEmail", handleSendTestRoomingEmail).Needs(AdminGetter)
	AddSessionHandler("/doSendRealRoomingEmail", handleSendRealRoomingEmail).Needs(AdminGetter)

	AddSessionHandler("/testUpdatesMail", handleTestSendUpdatesEmail).Needs(AdminGetter)
	AddSessionHandler("/sendUpdatesMail", handleAskSendUpdatesEmail).Needs(AdminGetter)
	AddSessionHandler("/doSendTestUpdatesEmail", handleSendTestUpdatesEmail).Needs(AdminGetter)
	AddSessionHandler("/doSendRealUpdatesEmail", handleSendRealUpdatesEmail).Needs(AdminGetter)

	AddSessionHandler("/testFinalEmail", handleTestSendFinalEmail).Needs(AdminGetter)

	AddSessionHandler("/info", handleInfo).Needs(PersonGetter)

	AddSessionHandler("/", handleIndex)
	//AddSessionHandler("/map", handleLoadMap).Needs(PersonGetter)

	rand.Seed(time.Now().UnixNano())

}

func handleIndex(ctx context.Context, wr WrappedRequest) {
	eventName := "PSR2021"
	if wr.Event != nil {
		eventName = wr.Event.ShortName
	}
	var tpl = template.Must(template.ParseFiles("templates/main.html", "templates/"+eventName+"/index.html"))
	if err := tpl.ExecuteTemplate(wr.ResponseWriter, "index.html", wr.TemplateData); err != nil {
		log.Println(err)
	}
}

func handleAdmin(ctx context.Context, wr WrappedRequest) {
	var tpl = template.Must(template.ParseFiles("templates/main.html", "templates/admin.html"))
	if err := tpl.ExecuteTemplate(wr.ResponseWriter, "admin.html", wr.TemplateData); err != nil {
		log.Println(err)
	}
}
