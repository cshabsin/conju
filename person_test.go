package conju

import (
	"fmt"
	"testing"
)

func ExampleFirstName() {
	p := Person{
		FirstName: "Christopher",
		Nickname:  "Chris",
		LastName:  "Shabsin",
	}
	fmt.Printf("Informal: %s\n", p.GetFirstName(Informal))
	fmt.Printf("Formal: %s\n", p.GetFirstName(Formal))
	// Output:
	// Informal: Chris
	// Formal: Christopher
}

func TestFullNameWithFormality(t *testing.T) {
	p := Person{
		FirstName: "Christopher",
		Nickname:  "Chris",
		LastName:  "Shabsin",
	}
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

