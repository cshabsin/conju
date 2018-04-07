package conju

import "fmt"

const SiteLink = "https://psr2018.shabsin.com"

func init() {
	AddSessionHandler("/increment", handleIncrement).Needs(PersonGetter)
	AddSessionHandler("/reloadData", AskReloadData).Needs(PersonGetter).Needs(AdminGetter)
	AddSessionHandler("/doReloadData", ReloadData).Needs(PersonGetter).Needs(AdminGetter)
	AddSessionHandler("/clearData", ClearAllData).Needs(PersonGetter).Needs(AdminGetter)

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
	AddSessionHandler("/rsvp", handleViewInvitationUser).Needs(InvitationGetter)

	AddSessionHandler("/sendMail", handleSendMail).Needs(InvitationGetter).Needs(AdminGetter)
	AddSessionHandler("/doSendMail", handleDoSendMail).Needs(InvitationGetter).Needs(AdminGetter)

	AddSessionHandler("/needsLogin", handleIncrement).Needs(InvitationGetter)
	AddSessionHandler("/checkLogin", checkLogin).Needs(InvitationGetter)
	AddSessionHandler("/resendInvitation", handleResendInvitation)
	AddSessionHandler(resentInvitationPage, handleResentInvitation)

	AddSessionHandler("/", handleHomePage)
}

func handleIncrement(wr WrappedRequest) {
	if wr.Values["n"] == nil {
		wr.SetSessionValue("n", 0)
	} else {
		wr.SetSessionValue("n", wr.Values["n"].(int)+1)
	}
	wr.SaveSession()
	ev := wr.Event
	var event_name string
	if ev != nil {
		event_name = ev.Name
	}
	wr.ResponseWriter.Write([]byte(
		fmt.Sprintf("%s\n%d\n", event_name, wr.Values["n"].(int))))
}
