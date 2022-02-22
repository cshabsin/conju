package invitation

// Each event should have a list of acceptable RSVP statuses
type RsvpStatus int

// TODO: move most of these to the datastore.

const (
	No RsvpStatus = iota
	Maybe
	FriSat
	ThuFriSat
	SatSun
	FriSatSun
	FriSatPlusEither
	WeddingOnly
	Fri
	Sat
	MealsOnly
)

type RsvpStatusInfo struct {
	Status           RsvpStatus
	ShortDescription string
	LongDescription  string
	Attending        bool
	Undecided        bool
	NoLodging        bool
	BaseCost         [6]float64
	AddOnCost        [6]float64
	Meals            []Meal
}

func GetAllRsvpStatuses() []RsvpStatusInfo {
	return []RsvpStatusInfo{
		{
			Status:           No,
			ShortDescription: "No",
			LongDescription:  "Will not attend",
			Attending:        false,
		},
		{
			Status:           Maybe,
			ShortDescription: "Maybe",
			LongDescription:  "Undecided",
			Attending:        false,
			Undecided:        true,
		},
		{
			Status:           FriSat,
			ShortDescription: "FriSat",
			LongDescription:  "Will attend: Friday - Sunday",
			Attending:        true,
			// TODO: make costs a property of the event
			BaseCost: [6]float64{0, 272.50, 176.58, 144.61, 128.62, 119.00},
			Meals:    []Meal{FriDin, SatBrk, SatLun, SatDin, SunBrk, SunLun},
		},
		{
			Status:           ThuFriSat,
			ShortDescription: "ThuFriSat",
			LongDescription:  "Will attend: Thursday - Sunday",
			Attending:        true,
			// TODO: make costs a property of the event
			BaseCost:  [6]float64{0, 272.50, 176.58, 144.61, 128.62, 119.00},
			AddOnCost: [6]float64{0, 124.26, 76.30, 60.31, 52.32, 47.52},
			Meals:     []Meal{FriBrk, FriLun, FriDin, SatBrk, SatLun, SatDin, SunBrk, SunLun},
		},
		{
			Status:           SatSun,
			ShortDescription: "SatSun",
			LongDescription:  "Will attend: Saturday - Sunday",
			Attending:        true,
		},
		{
			Status:           FriSatSun,
			ShortDescription: "FriSatSun",
			LongDescription:  "Will attend: Friday - Sunday",
			Attending:        true,
		},
		{
			Status:           FriSatPlusEither,
			ShortDescription: "FriSatPlusEither",
			LongDescription:  "Will attend: Friday - Sunday, plus either Thursday or Sunday nights",
			Attending:        true,
		},
		{
			Status:           WeddingOnly,
			ShortDescription: "WeddingOnly",
			LongDescription:  "Will attend: Wedding Only (no overnights)",
			Attending:        true,
			NoLodging:        true,
		},
		{
			Status:           Fri,
			ShortDescription: "Fri",
			LongDescription:  "Will attend: Friday - Saturday",
			Attending:        true,
		},
		{
			Status:           Sat,
			ShortDescription: "Sat",
			LongDescription:  "Will attend: Saturday - Sunday",
			Attending:        true,
		},
		{
			Status:           MealsOnly,
			ShortDescription: "Meals",
			LongDescription:  "Will need meals but not lodging",
			Attending:        true,
			NoLodging:        true,
		},
	}
}
