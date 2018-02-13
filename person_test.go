package conju

import (
	"fmt"
)

func ExampleFirstName() {
	p := Person{
		FirstName: "Christopher",
		Nickname: "Chris",
		LastName: "Shabsin",
	}
	fmt.Printf("Informal: %s\n", p.GetFirstName(Informal))
	fmt.Printf("Formal: %s\n", p.GetFirstName(Formal))
	// Output:
	// Informal: Chris
	// Formal: Christopher
}
