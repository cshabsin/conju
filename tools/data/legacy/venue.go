package legacy

type Venue struct {
	Name          string
	ShortName     string
	ContactPerson string
	ContactPhone  string
	ContactEmail  string
	Website       string
}

func (v *Venue) Kind() string {
	return "Venue"
}
