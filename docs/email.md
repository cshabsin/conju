# Email notes

## Email template objects

Events have Email templates associated with them. For example, the PSR 2018
event might have a "Save the Date" template. Email templates can be evaluated
in the context of a person, or in the context of an invitation (see below for
the data structure passed into such templates).

## Supported operations

* Create
* Edit
* Copy within event
* Copy between events
* Send to person
* Send to personset (TODO: what is a personset?)
* Send to invitation
* Send to invitation set

## Template Data Structures

### All templates

All template handlers fill the current event from the session.

```go
struct {
	CurrentEvent Event
}
```

These fields can be accessed in the 


### Person-centric templates

