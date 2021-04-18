package conju

type Activity struct {
	Keyword     string
	Description string
	NeedsLeader bool
}

type ActivityWithKey struct {
	EncodedKey string
	Activity   Activity
}
