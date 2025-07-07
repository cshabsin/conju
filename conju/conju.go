package conju

import (
	"context"
	"html/template"
	"log"
	"mime"
	"net/http"
	"path/filepath"
	"strings"

	"cloud.google.com/go/datastore"
	"github.com/cshabsin/conju/conju/dsclient"
)

func Register(s Sessionizer) {
	s.AddSessionHandler("/_ah/warmup", handleWarmup)

	s.AddSessionHandler("/reloadData", AskReloadData).Needs(AdminGetter)
	s.AddSessionHandler("/doReloadData", ReloadData).Needs(AdminGetter)
	//AddSessionHandler("/clearData", ClearAllData).Needs(AdminGetter)
	s.AddSessionHandler("/repairData", RepairData).Needs(AdminGetter)

	s.AddSessionHandler("/reloadHousingSetup", AskReloadHousingSetup).Needs(AdminGetter)
	s.AddSessionHandler("/doReloadHousingSetup", ReloadHousingSetup).Needs(AdminGetter)
	//	AddSessionHandler("/clearHousingSetup", ClearAllHousingSetup).Needs(AdminGetter)

	s.AddSessionHandler("/login", handleLogin("/rsvp"))
	s.AddSessionHandler(loginErrorPage, handleLoginError)
	s.AddSessionHandler("/logout", handleLogout)
	s.AddSessionHandler("/resendInvitation", handleResendInvitation)
	s.AddSessionHandler(resentInvitationPage, handleResentInvitation)

	s.AddSessionHandler("/admin", handleAdmin).Needs(PersonGetter).Needs(AdminGetter)

	s.AddSessionHandler("/listPeople", handleListPeople).Needs(PersonGetter).Needs(AdminGetter)
	s.AddSessionHandler("/updatePersonForm", handleUpdatePersonForm).Needs(PersonGetter).Needs(AdminGetter)
	s.AddSessionHandler("/saveUpdatePerson", handleSaveUpdatePerson).Needs(PersonGetter).Needs(AdminGetter)

	s.AddSessionHandler("/invitations", handleInvitations).Needs(PersonGetter).Needs(AdminGetter)
	s.AddSessionHandler("/receivePay", handleReceivePay).Needs(PersonGetter).Needs(AdminGetter)
	s.AddSessionHandler("/doReceivePay", handleDoReceivePay).Needs(PersonGetter).Needs(AdminGetter)
	s.AddSessionHandler("/copyInvitations", handleCopyInvitations).Needs(PersonGetter).Needs(AdminGetter)
	s.AddSessionHandler("/addInvitation", handleAddInvitation).Needs(PersonGetter).Needs(AdminGetter)
	s.AddSessionHandler("/deleteInvitation", handleDeleteInvitation).Needs(PersonGetter).Needs(AdminGetter)
	s.AddSessionHandler("/viewInvitation", handleViewInvitationAdmin).Needs(PersonGetter).Needs(AdminGetter)
	s.AddSessionHandler("/saveInvitation", handleSaveInvitation).Needs(InvitationGetter)

	s.AddSessionHandler("/events", handleEvents).Needs(PersonGetter).Needs(AdminGetter)
	s.AddSessionHandler("/createUpdateEvent", handleCreateUpdateEvent).Needs(PersonGetter).Needs(AdminGetter)

	s.AddSessionHandler("/rsvp", handleViewInvitationUser).Needs(InvitationGetter)

	s.AddSessionHandler("/rsvpReport", handleRsvpReport).Needs(PersonGetter).Needs(AdminGetter)
	s.AddSessionHandler("/activitiesReport", handleActivitiesReport).Needs(PersonGetter).Needs(AdminGetter)
	s.AddSessionHandler("/roomingReport", handleRoomingReport).Needs(PersonGetter).Needs(AdminGetter)
	s.AddSessionHandler("/handleSaveReservations", handleSaveReservations).Needs(PersonGetter).Needs(AdminGetter)
	s.AddSessionHandler("/foodReport", handleFoodReport).Needs(PersonGetter).Needs(AdminGetter)
	s.AddSessionHandler("/ridesReport", handleRidesReport).Needs(PersonGetter).Needs(AdminGetter)

	s.AddSessionHandler("/rooming", handleRoomingTool).Needs(PersonGetter).Needs(AdminGetter)
	s.AddSessionHandler("/saveRooming", handleSaveRooming).Needs(PersonGetter).Needs(AdminGetter)

	s.AddSessionHandler("/viewMyInvitation", handleViewMyInvitation).Needs(InvitationGetter)

	s.AddSessionHandler("/sendMail", handleSendMail).Needs(InvitationGetter).Needs(AdminGetter)
	s.AddSessionHandler("/editMail", handleEditMail).Needs(InvitationGetter).Needs(AdminGetter)
	s.AddSessionHandler("/doSendMail", handleDoSendMail).Needs(InvitationGetter).Needs(AdminGetter)
	s.AddSessionHandler("/previewMailTemplate", handlePreviewMailTemplate).Needs(AdminGetter)
	s.AddSessionHandler("/saveMail", handleSaveMail).Needs(InvitationGetter).Needs(AdminGetter)

	s.AddSessionHandler("/testRoomingMail", handleTestSendRoomingEmail).Needs(AdminGetter)
	s.AddSessionHandler("/sendRoomingMail", handleAskSendRoomingEmail).Needs(AdminGetter)
	s.AddSessionHandler("/doSendTestRoomingEmail", handleSendTestRoomingEmail).Needs(AdminGetter)
	s.AddSessionHandler("/doSendRealRoomingEmail", handleSendRealRoomingEmail).Needs(AdminGetter)

	s.AddSessionHandler("/testUpdatesMail", handleTestSendUpdatesEmail).Needs(AdminGetter)
	s.AddSessionHandler("/sendUpdatesMail", handleAskSendUpdatesEmail).Needs(AdminGetter)
	s.AddSessionHandler("/doSendTestUpdatesEmail", handleSendTestUpdatesEmail).Needs(AdminGetter)
	s.AddSessionHandler("/doSendRealUpdatesEmail", handleSendRealUpdatesEmail).Needs(AdminGetter)

	s.AddSessionHandler("/testFinalEmail", handleTestSendFinalEmail).Needs(AdminGetter)

	s.AddSessionHandler("/info", handleInfo).Needs(PersonGetter)

	s.AddSessionHandler("/dbmedia/", handleDBMedia).Needs(PersonGetter)

	//AddSessionHandler("/map", handleLoadMap).Needs(PersonGetter)

	s.AddSessionHandler("/", handleIndex).Needs(PersonGetter)
}

func handleIndex(ctx context.Context, wr *WrappedRequest) {
	eventName := "PSR2025"
	if wr.Event != nil {
		eventName = wr.Event.ShortName
	}
	var tpl = template.Must(template.ParseFiles("templates/main.html", "templates/"+eventName+"/index.html"))
	if err := tpl.ExecuteTemplate(wr.ResponseWriter, "index.html", wr.TemplateData); err != nil {
		log.Println(err)
	}
}

func handleAdmin(ctx context.Context, wr *WrappedRequest) {
	var tpl = template.Must(template.ParseFiles("templates/main.html", "templates/admin.html"))
	if err := tpl.ExecuteTemplate(wr.ResponseWriter, "admin.html", wr.TemplateData); err != nil {
		log.Println(err)
	}
}

func handleDBMedia(ctx context.Context, wr *WrappedRequest) {
	log.Printf("handleDBMedia: %s", wr.URL.Path)
	client := dsclient.FromContext(ctx)
	if client == nil {
		log.Println("Datastore client is not available in context")
		http.Error(wr.ResponseWriter, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	path := strings.TrimPrefix(wr.URL.Path, "/dbmedia/")
	key := datastore.NameKey("Media", path, nil)
	var media DBMedia
	if err := client.Get(ctx, key, &media); err != nil {
		log.Printf("Error retrieving media %s: %v", path, err)
		http.Error(wr.ResponseWriter, "Media not found", http.StatusNotFound)
		return
	}

	wr.ResponseWriter.Header().Set("Content-Type", mime.TypeByExtension(filepath.Ext(path)))
	wr.ResponseWriter.Write([]byte(media.Value))
}

type DBMedia struct {
	Name  string `datastore:"name"`
	Value []byte `datastore:"value,noindex"`
}
