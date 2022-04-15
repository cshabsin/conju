package conju

import "github.com/cshabsin/conju/invitation"

// Each event should have a list of acceptable RSVP statuses
type RsvpStatus int

// TODO: move most of these to the datastore.

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
	Meals            []invitation.Meal
}

type PaymentType int

const (
	Cash = iota
	Check
	GoogleWallet
	Venmo
	PayPal
)

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
	MonitorRange HousingPreferenceBoolean = iota
	CloseBuilding
	FarBuilding
	CanCrossRoad
	PreferFar
	FartherBuilding
	ShareBed
	WillingExpensive
	PreferExpensive
)

type HousingPreferenceBooleanType int

const (
	Desired HousingPreferenceBooleanType = iota
	Acceptable
)

type HousingPreferenceBooleanInfo struct {
	Boolean                   HousingPreferenceBoolean
	Name                      string
	SinglePersonDescription   string
	MultiplePeopleDescription string
	CoupleDescription         string
	ReportDescription         string
	SupplementalInfo          string
	ForChildren               bool
	ForMultiples              bool
	Bit                       int
	PreferenceType            HousingPreferenceBooleanType
	Hidden                    bool
}

func GetAllHousingPreferenceBooleans() []HousingPreferenceBooleanInfo {
	toReturn := []HousingPreferenceBooleanInfo{
		{
			Boolean:                   MonitorRange,
			Name:                      "MonitorRange",
			MultiplePeopleDescription: "We would prefer to be within baby-monitor range of the main common room.",
			ReportDescription:         "Monitor Range",
			ForChildren:               true,
			Bit:                       64,
			PreferenceType:            Desired,
		},
		{
			Boolean:                   CloseBuilding,
			Name:                      "CloseBuilding",
			MultiplePeopleDescription: "We can stay in a building that is not within baby-monitor range of the main common room, but is very close by.",
			ReportDescription:         "Close Building",
			ForChildren:               true,
			Bit:                       32,
			PreferenceType:            Acceptable,
		},
		{
			Boolean:                   FarBuilding,
			Name:                      "FarBuilding",
			MultiplePeopleDescription: "We can stay in a building that is ~100 yards away from the main common room.",
			ReportDescription:         "Far Building",
			ForChildren:               true,
			Bit:                       16,
			PreferenceType:            Acceptable,
		},
		{
			Boolean:                   CanCrossRoad,
			Name:                      "CanCrossRoad",
			MultiplePeopleDescription: "Everyone in our party can cross a (low-traffic) road, alone, safely, even at night.",
			ReportDescription:         "Across Road",
			ForChildren:               true,
			Bit:                       8,
			PreferenceType:            Acceptable,
			Hidden:                    true,
		},
		{
			Boolean:                   PreferFar,
			Name:                      "PreferFar",
			MultiplePeopleDescription: "We would prefer to be housed far from the main common room.",
			SinglePersonDescription:   "I would prefer to be housed far from the main common room.",
			ReportDescription:         "Prefer Farther",
			Bit:                       4,
			PreferenceType:            Desired,
		},
		{
			Boolean:                   FartherBuilding,
			Name:                      "FartherBuilding",
			MultiplePeopleDescription: "In case of overflow, we would be willing to be housed in a building that is outside of our main cluster of buildings.",
			SinglePersonDescription:   "In case of overflow, I would be willing to be housed in a building that is outside of our main cluster of buildings.",
			SupplementalInfo:          "Other buildings are more expensive, but are correspondingly nicer, and you may want a car to get back and forth (about half a mile).",
			ReportDescription:         "Farther Building Okay",
			Bit:                       1,
			PreferenceType:            Acceptable,
			Hidden:                    true,
		},
		{
			Boolean:                   WillingExpensive,
			Name:                      "WillingExpensive",
			MultiplePeopleDescription: "We would be willing to stay in nicer, more expensive (+~$50/night) housing that is ~300 yards away from the main common room.",
			SinglePersonDescription:   "I would be willing to stay in nicer, more expensive (+~$50/night) housing that is ~300 yards away from the main common room.",
			ReportDescription:         "Expensive Housing Okay",
			Bit:                       128,
			PreferenceType:		   Acceptable,
		},
		{
			Boolean:                   PreferExpensive,
			Name:                      "PreferExpensive",
			MultiplePeopleDescription: "We would prefer to stay in nicer, more expensive (+~$50/night) housing that is ~300 yards away from the main common room.",
			SinglePersonDescription:   "I would prefer to stay in nicer, more expensive (+~$50/night) housing that is ~300 yards away from the main common room.",
			ReportDescription:         "Expensive Housing Preferred",
			Bit:                       256,
			PreferenceType:		   Desired,
		},
		{
			Boolean:                   ShareBed,
			Name:                      "ShareBed",
			MultiplePeopleDescription: "We would prefer a room with a bed that sleeps 2.",
			CoupleDescription:         "We would prefer to share a bed.",
			ReportDescription:         "Share Bed",
			ForMultiples:              true,
			Bit:                       2,
			PreferenceType:            Desired,
		},
	}

	return toReturn
}

func GetPreferenceTypeMask(preferenceType HousingPreferenceBooleanType) int {
	mask := 0
	for _, info := range GetAllHousingPreferenceBooleans() {
		if info.PreferenceType == preferenceType {
			mask += info.Bit
		}
	}
	return mask
}

func GetAdultPreferenceMask() int {
	mask := 0
	for _, info := range GetAllHousingPreferenceBooleans() {
		if info.ForChildren && info.PreferenceType == Acceptable {
			mask += info.Bit
		}
	}
	return mask
}

type DrivingPreference int

const (
	DrivingNotSet = iota
	NoCarpool
	Driving
	Riding
	DriveIfNeeded
)

type DrivingPreferenceInfo struct {
	Preference                DrivingPreference
	SinglePersonDescription   string
	MultiplePeopleDescription string
	CoupleDescription         string
	ReportDescription         string
}

func GetAllDrivingPreferences() []DrivingPreferenceInfo {
	var toReturn []DrivingPreferenceInfo

	toReturn = append(toReturn, DrivingPreferenceInfo{
		Preference:                DrivingNotSet,
		SinglePersonDescription:   "-- Select your ride-sharing preferences --",
		MultiplePeopleDescription: "-- Select your ride-sharing preferences --",
		ReportDescription:         "Not Set",
	})
	toReturn = append(toReturn, DrivingPreferenceInfo{
		Preference:                NoCarpool,
		SinglePersonDescription:   "I will drive by myself.",
		MultiplePeopleDescription: "We will drive by ourselves.",
		ReportDescription:         "Alone",
	})
	toReturn = append(toReturn, DrivingPreferenceInfo{
		Preference:                Driving,
		SinglePersonDescription:   "I will have some extra room in my car and would love company.",
		MultiplePeopleDescription: "We will have some extra room in our car and would love company.",
		ReportDescription:         "Driving",
	})
	toReturn = append(toReturn, DrivingPreferenceInfo{
		Preference:                Riding,
		SinglePersonDescription:   "I will need a ride.",
		MultiplePeopleDescription: "We will need a ride.",
		ReportDescription:         "Riding",
	})
	toReturn = append(toReturn, DrivingPreferenceInfo{
		Preference:                DriveIfNeeded,
		SinglePersonDescription:   "I could drive but would rather ride.",
		MultiplePeopleDescription: "We could drive but would rather ride.",
		ReportDescription:         "Either",
	})

	return toReturn
}

type ParkingType int

const (
	NoElectric = iota
	PluginHybrid
	PureElectric
)

type ParkingTypeInfo struct {
	Parking                   ParkingType
	SinglePersonDescription   string
	MultiplePeopleDescription string
	ReportDescription         string
}

func GetAllParkingTypes() []ParkingTypeInfo {
	var toReturn []ParkingTypeInfo

	toReturn = append(toReturn, ParkingTypeInfo{
		Parking:                   NoElectric,
		SinglePersonDescription:   "My vehicle doesn't need to be charged.",
		MultiplePeopleDescription: "Our vehicle doesn't need to be charged.",
		ReportDescription:         "No Electricity Needed",
	})
	toReturn = append(toReturn, ParkingTypeInfo{
		Parking:                   PluginHybrid,
		SinglePersonDescription:   "I have a plug-in hybrid and would prefer to charge it at some point over the weekend.",
		MultiplePeopleDescription: "We have a plug-in hybrid and would prefer to charge it at some point over the weekend.",
		ReportDescription:         "Want Electric",
	})
	toReturn = append(toReturn, ParkingTypeInfo{
		Parking:                   PureElectric,
		SinglePersonDescription:   "I have a fully-electric vehicle and will need to charge it at some point over the weekend.",
		MultiplePeopleDescription: "We have a fully-electric vehicle and will need to charge it at some point over the weekend.",
		ReportDescription:         "Need Electric",
	})

	return toReturn
}

type ActivityRanking int

const (
	ActivityNotSet = iota
	ActivityNo
	ActivityMaybe
	ActivityDefinitely
)
