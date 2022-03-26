package main

import (
	"google.golang.org/appengine"

	"github.com/cshabsin/conju/conju"
	"github.com/cshabsin/conju/view/poll"
)

func main() {
	conju.Register()
	poll.Register()

	appengine.Main()
}
