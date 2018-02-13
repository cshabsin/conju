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
	fn := CollectiveAddress([]Person{chris}, Informal)
	if fn != "Chris Shabsin" {
		t.Errorf("Single person with nickname, informal was incorrect, got: %s, want: %s.", fn, "Chris Shabsin")
	}

	fn = CollectiveAddress([]Person{dana, chris}, Informal)
	if fn != "Dana Scott & Chris Shabsin" {
		t.Errorf("Couple different names, informal was incorrect, got: %s, want: %s.", fn, "Dana Scott & Chris Shabsin")
	}

	fn = CollectiveAddress([]Person{chris, lydia}, Formal)
	if fn != "Christopher & Lydia Shabsin" {
		t.Errorf("Couple same name, formal was incorrect, got: %s, want: %s.", fn, "Christopher & Lydia Shabsin")
	}

	fn = CollectiveAddress([]Person{chris, dana, lydia}, Informal)
	if fn != "Chris, Dana & Lydia" {
		t.Errorf(">2 different names, informal was incorrect, got: %s, want: %s.", fn, "Chris, Dana & Lydia")
	}

	fn = CollectiveAddress([]Person{rick, chris, lydia}, Formal)
	if fn != "Richard, Christopher & Lydia Shabsin" {
		t.Errorf(">2 different names, formal was incorrect, got: %s, want: %s.", fn, "Richard, Christopher & Lydia Shabsin")
	}

	fn = CollectiveAddress([]Person{rick, chris, dana, lydia}, Informal)
	if fn != "Rick, Chris, Dana & Lydia" {
		t.Errorf(">3 different names, informal was incorrect, got: %s, want: %s.", fn, "Rick, Chris, Dana & Lydia")
	}

}
