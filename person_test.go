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
	p := chris

	fn := p.FullNameWithFormality(Full)
	if fn != "Christopher (Chris) Shabsin" {
		t.Errorf("Full name was incorrect, got: %s, want: %s.", fn, "Christopher (Chris) Shabsin")
	}
	fn = p.FullNameWithFormality(Formal)
	if fn != "Christopher Shabsin" {
		t.Errorf("Formal name was incorrect, got: %s, want: %s.", fn, "Christopher Shabsin")
	}
	fn = p.FullNameWithFormality(Informal)
	if fn != "Chris Shabsin" {
		t.Errorf("Formal name was incorrect, got: %s, want: %s.", fn, "Chris Shabsin")
	}
}

func TestCollectiveAddress(t *testing.T) {
	fn1 := CollectiveAddress([]Person{chris}, Informal)
	if fn1 != "Chris Shabsin" {
		t.Errorf("Single person with nickname, informal was incorrect, got: %s, want: %s.", fn1, "Chris Shabsin")
	}

	fn2 := CollectiveAddress([]Person{dana, chris}, Informal)
	if fn2 != "Dana Scott & Chris Shabsin" {
		t.Errorf("Couple different names, informal was incorrect, got: %s, want: %s.", fn2, "Dana Scott & Chris Shabsin")
	}

	fn3 := CollectiveAddress([]Person{chris, lydia}, Formal)
	if fn3 != "Christopher & Lydia Shabsin" {
		t.Errorf("Couple same name, formal was incorrect, got: %s, want: %s.", fn3, "Christopher & Lydia Shabsin")
	}

	fn4 := CollectiveAddress([]Person{chris, dana, lydia}, Informal)
	if fn4 != "Chris, Dana & Lydia" {
		t.Errorf(">2 different names, informal was incorrect, got: %s, want: %s.", fn4, "Chris, Dana & Lydia")
	}

	fn5 := CollectiveAddress([]Person{rick, chris, lydia}, Formal)
	if fn5 != "Richard, Christopher & Lydia Shabsin" {
		t.Errorf(">2 different names, formal was incorrect, got: %s, want: %s.", fn5, "Richard, Christopher & Lydia Shabsin")
	}

	fn6 := CollectiveAddress([]Person{rick, chris, dana, lydia}, Informal)
	if fn6 != "Rick, Chris, Dana & Lydia" {
		t.Errorf(">3 different names, informal was incorrect, got: %s, want: %s.", fn6, "Rick, Chris, Dana & Lydia")
	}



}

