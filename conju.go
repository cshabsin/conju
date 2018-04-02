package conju

import "fmt"

func init() {
	AddSessionHandler("/increment", handleIncrement).Needs(EventGetter)
	AddSessionHandler("/reloadData", AskReloadData).Needs(AdminGetter)
	AddSessionHandler("/doReloadData", ReloadData).Needs(AdminGetter)
	AddSessionHandler("/clearData", ClearAllData).Needs(AdminGetter)

	AddSessionHandler("/listPeople", handleListPeople).Needs(EventGetter).Needs(AdminGetter)
	AddSessionHandler("/updatePersonForm", handleUpdatePersonForm).Needs(EventGetter).Needs(AdminGetter)
	AddSessionHandler("/saveUpdatePerson", handleSaveUpdatePerson).Needs(AdminGetter)
	AddSessionHandler("/invitations", handleInvitations).Needs(EventGetter).Needs(AdminGetter)
	AddSessionHandler("/copyInvitations", handleCopyInvitations).Needs(EventGetter).Needs(AdminGetter)
	AddSessionHandler("/addInvitation", handleAddInvitation).Needs(EventGetter).Needs(AdminGetter)
	AddSessionHandler("/viewInvitation", handleViewInvitation).Needs(EventGetter)
	AddSessionHandler("/saveInvitation", handleSaveInvitation).Needs(EventGetter)

	AddSessionHandler("/login", handleLogin).Needs(EventGetter)

	AddSessionHandler("/needsLogin", handleIncrement).Needs(LoginGetter)
	AddSessionHandler("/checkLogin", checkLogin).Needs(LoginGetter)
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
