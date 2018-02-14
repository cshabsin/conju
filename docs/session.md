# Session Notes

There are two types of login state to be tracked, perhaps completely
independently.

## Administrator Login

Access to admin consoles and email interfaces. Keyed off of OAuth
login to known Gmail accounts.

We also need to store an OAuth token to send emails with, using the
Gmail API. Separate?

## Guest Login

### Invitation Conundrum

Based off of invitation code, in the current auth scheme. In the new
data model, as envisioned, the invitation code will give access to all
the Person objects associated with the code. Includes write access to
update info?

Invitation codes now can be associated with multiple Invitation
objects, for different Events. Need to be able to switch events?

Consider associating Persons with Gmail accounts, to allow people to
log in with Gmail and access the Invitations associated with that
Person.

If we create an Invitation that contains a full family, does that give
everyone in the family access to all the other family members' Person
table data? How about dating couples? If a couple breaks up, can they
still log in to the old event to see the other person's data?

Invitation may be too dangerous to keep as currently "designed", for
anything longer lived than a single event.