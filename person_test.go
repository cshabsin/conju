package conju

import (
	"fmt"
	"testing"
)

var chris = Person{
	FirstName: "Christopher",
	Nickname:  "Chris",
	LastName:  "Shabsin",
}
var dana = Person{
	FirstName: "Dana",
	LastName:  "Scott",
}
var lydia = Person{
	FirstName: "Lydia",
	LastName:  "Shabsin",
}
var rick = Person{
	FirstName: "Richard",
	Nickname:  "Rick",
	LastName:  "Shabsin",
}

func ExampleFirstName() {
	fmt.Printf("Informal: %s\n", chris.GetFirstName(Informal))
	fmt.Printf("Formal: %s\n", chris.GetFirstName(Formal))
	// Output:
	// Informal: Chris
	// Formal: Christopher
}

func TestFullNameWithFormality(t *testing.T) {
	fn := chris.FullNameWithFormality(Full)
	if fn != "Christopher (Chris) Shabsin" {
		t.Errorf("Full name was incorrect, got: %s, want: %s.", fn, "Christopher (Chris) Shabsin")
	}
	fn = chris.FullNameWithFormality(Formal)
	if fn != "Christopher Shabsin" {
		t.Errorf("Formal name was incorrect, got: %s, want: %s.", fn, "Christopher Shabsin")
	}
	fn = chris.FullNameWithFormality(Informal)
	if fn != "Chris Shabsin" {
		t.Errorf("Formal name was incorrect, got: %s, want: %s.", fn, "Chris Shabsin")
	}
}

func TestCollectiveAddress(t *testing.T) {
	type TestCase struct {
		Name string
		People []Person
		NameFormality
		Want string
	}
	testcases := []TestCase{
		{
			Name: "Single Person w/ Nickname",
			People: []Person{chris},
			NameFormality: Informal,
			Want: "Chris Shabsin",
		},
		{
			Name: "Couple w/ Different Last Names",
			People: []Person{dana, chris},
			NameFormality: Informal,
			Want: "Dana Scott & Chris Shabsin",
		},
		{
			Name: "Couple w/ Same Last Name",
			People: []Person{chris, lydia},
			NameFormality: Formal,
			Want: "Christopher & Lydia Shabsin",
		},
		{
			Name: ">2 Different Names, Informal",
			People: []Person{chris, dana, lydia},
			NameFormality: Informal,
			Want: "Chris, Dana & Lydia",
		},
		{
			Name: ">2 Different Names, Formal",
			People: []Person{chris, dana, lydia},
			NameFormality: Formal,
			Want: "Christopher, Dana & Lydia",
		},
		{
			Name: ">2 Same Names, Formal",
			People: []Person{rick, chris, lydia},
			NameFormality: Formal,
			Want: "Richard, Christopher & Lydia Shabsin",
		},
		{
			Name: ">3 Different Names, Informal",
			People: []Person{rick, chris, dana, lydia},
			NameFormality: Informal,
			Want: "Rick, Chris, Dana & Lydia",
		},
	}
	for _, tc := range(testcases) {
		fn := CollectiveAddress(tc.People, tc.NameFormality)
		if fn != tc.Want {
			t.Errorf("%s incorrect, got: %s, want: %s.", tc.Name, fn, tc.Want)
		}
	}
}
