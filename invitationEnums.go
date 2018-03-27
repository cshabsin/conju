package conju

// Each event should have a list of acceptable RSVP statuses
type RsvpStatus int

const (
	No = iota
	Maybe
	FriSat
	ThuFriSat
	SatSun
	FriSatSun
	FriSatPlusEither
	WeddingOnly
	Fri
	Sat
)

type RsvpStatusInfo struct {
	Status           RsvpStatus
	ShortDescription string
	LongDescription  string
	Attending        bool
	Undecided        bool
	NoLodging        bool
}

func GetAllRsvpStatuses() []RsvpStatusInfo {
	var toReturn []RsvpStatusInfo

	toReturn = append(toReturn, RsvpStatusInfo{
		Status:           No,
		ShortDescription: "No",
		LongDescription:  "Will not attend",
		Attending:        false,
	})
	toReturn = append(toReturn, RsvpStatusInfo{
		Status:           Maybe,
		ShortDescription: "Maybe",
		LongDescription:  "Undecided",
		Attending:        false,
		Undecided:        true,
	})
	toReturn = append(toReturn, RsvpStatusInfo{
		Status:           FriSat,
		ShortDescription: "FriSat",
		LongDescription:  "Will attend: Friday - Sunday",
		Attending:        true,
	})
	toReturn = append(toReturn, RsvpStatusInfo{
		Status:           ThuFriSat,
		ShortDescription: "ThuFriSat",
		LongDescription:  "Will attend: Thursday - Sunday",
		Attending:        true,
	})
	toReturn = append(toReturn, RsvpStatusInfo{
		Status:           SatSun,
		ShortDescription: "SatSun",
		LongDescription:  "Will attend: Saturday - Sunday",
		Attending:        true,
	})
	toReturn = append(toReturn, RsvpStatusInfo{
		Status:           FriSatSun,
		ShortDescription: "FriSatSun",
		LongDescription:  "Will attend: Friday - Sunday",
		Attending:        true,
	})
	toReturn = append(toReturn, RsvpStatusInfo{
		Status:           FriSatPlusEither,
		ShortDescription: "FriSatPlusEither",
		LongDescription:  "Will attend: Friday - Sunday, plus either Thursday or Sunday nights",
		Attending:        true,
	})
	toReturn = append(toReturn, RsvpStatusInfo{
		Status:           WeddingOnly,
		ShortDescription: "WeddingOnly",
		LongDescription:  "Will attend: Wedding Only (no overnights)",
		Attending:        true,
		NoLodging:        true,
	})
	toReturn = append(toReturn, RsvpStatusInfo{
		Status:           Fri,
		ShortDescription: "Fri",
		LongDescription:  "Will attend: Friday - Saturday",
		Attending:        true,
	})
	toReturn = append(toReturn, RsvpStatusInfo{
		Status:           Sat,
		ShortDescription: "Sat",
		LongDescription:  "Will attend: Saturday - Sunday",
		Attending:        true,
	})
	return toReturn
}

type HousingPreference int

const (
	HousingNotSet = iota
	NoRoommates
	SpecificRoommates
	KnownRoommates
	AnyRoommates
)

type HousingPreferenceInfo struct {
	Preference                HousingPreference
	SinglePersonDescription   string
	MultiplePeopleDescription string
	ReportDescription         string
}

func GetAllHousingPreferences() []HousingPreferenceInfo {
	var toReturn []HousingPreferenceInfo

	toReturn = append(toReturn, HousingPreferenceInfo{
		Preference:                HousingNotSet,
		SinglePersonDescription:   "-- Select your rooming preference --",
		MultiplePeopleDescription: "-- Select your rooming preference --",
		ReportDescription:         "not set",
	})
	toReturn = append(toReturn, HousingPreferenceInfo{
		Preference:                NoRoommates,
		SinglePersonDescription:   "I need a room to myself.",
		MultiplePeopleDescription: "We need a room to ourselves.",
		ReportDescription:         "no one",
	})
	toReturn = append(toReturn, HousingPreferenceInfo{
		Preference:                SpecificRoommates,
		SinglePersonDescription:   "I am willing to share a room with specific people, listed below.",
		MultiplePeopleDescription: "We are willing to share a room with specific people, listed below.",
		ReportDescription:         "specific people",
	})
	toReturn = append(toReturn, HousingPreferenceInfo{
		Preference:                KnownRoommates,
		SinglePersonDescription:   "I am willing to share a room with people I know.",
		MultiplePeopleDescription: "We are willing to share a room with people we know.",
		ReportDescription:         "known people",
	})
	toReturn = append(toReturn, HousingPreferenceInfo{
		Preference:                AnyRoommates,
		SinglePersonDescription:   "I am willing to share a room with anyone who needs a roommate.",
		MultiplePeopleDescription: "We are willing to share a room with anyone who needs a roommate.",
		ReportDescription:         "anyone",
	})

	return toReturn
}

type HousingPreferenceBoolean int

const (
	MonitorRange = iota
	CloseBuilding
	FarBuilding
	CanCrossRoad
	PreferFar
	ShareBed
	FartherBuilding
)

type HousingPreferenceBooleanInfo struct {
	Boolean                   HousingPreferenceBoolean
	SinglePersonDescription   string
	MultiplePeopleDescription string
	CoupleDescription         string
	ReportDescription         string
	SupplementalInfo          string
	ForChildren               bool
	ForMultiples              bool
	Bit                       int
}

func GetAllHousingPreferenceBooleans() []HousingPreferenceBooleanInfo {
	var toReturn []HousingPreferenceBooleanInfo

	toReturn = append(toReturn, HousingPreferenceBooleanInfo{
		Boolean:                   MonitorRange,
		MultiplePeopleDescription: "We would prefer to be within baby-monitor range of the main common room.",
		ReportDescription:         "Monitor Range",
		ForChildren:               true,
		Bit:                       64,
	})
	toReturn = append(toReturn, HousingPreferenceBooleanInfo{
		Boolean:                   CloseBuilding,
		MultiplePeopleDescription: "We can stay in a building that is not within baby-monitor range of the main common room, but is very close by.",
		ReportDescription:         "Close Building",
		ForChildren:               true,
		Bit:                       32,
	})
	toReturn = append(toReturn, HousingPreferenceBooleanInfo{
		Boolean:                   FarBuilding,
		MultiplePeopleDescription: "We can stay in a building that is ~100 yards away from the main common room.",
		ReportDescription:         "Far Building",
		ForChildren:               true,
		Bit:                       16,
	})
	toReturn = append(toReturn, HousingPreferenceBooleanInfo{
		Boolean:                   CanCrossRoad,
		MultiplePeopleDescription: "Everyone in our party can cross a (low-traffic) road, alone, safely, even at night.",
		ReportDescription:         "Across Road",
		ForChildren:               true,
		Bit:                       8,
	})
	toReturn = append(toReturn, HousingPreferenceBooleanInfo{
		Boolean:                   PreferFar,
		MultiplePeopleDescription: "We would prefer to be housed far from the main common room.",
		SinglePersonDescription:   "I would prefer to be housed far from the main common room.",
		ReportDescription:         "Prefer Farther",
		Bit:                       4,
	})
	toReturn = append(toReturn, HousingPreferenceBooleanInfo{
		Boolean:                   FartherBuilding,
		MultiplePeopleDescription: "In case of overflow, we would be willing to be housed in a building that is outside of our main cluster of buildings.",
		SinglePersonDescription:   "In case of overflow, I would be willing to be housed in a building that is outside of our main cluster of buildings.",
		SupplementalInfo:          "Other buildings are more expensive, but are correspondingly nicer, and you may want a car to get back and forth (about a half mile).",
		ReportDescription:         "Farther Building Okay",
		Bit:                       1,
	})
	toReturn = append(toReturn, HousingPreferenceBooleanInfo{
		Boolean:                   ShareBed,
		MultiplePeopleDescription: "We would prefer a room with a bed that sleeps 2.",
		CoupleDescription:         "We would prefer to share a bed.",
		ReportDescription:         "Share Bed",
		ForMultiples:              true,
		Bit:                       2,
	})

	return toReturn
}
