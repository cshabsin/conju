# Schema Notes

* Person
  * Contains:
    * name
    * contact info
    * birthdate/age
* Event
  * Contains:
    * name
    * start/end dates
  * Is Ancestor Of:
    * Invitation
      * Contains:
      	* slice of Key to Person objects
	* invitation code
