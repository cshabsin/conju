package main

import (
	"google.golang.org/appengine"

	_ "github.com/cshabsin/conju/conju"
	_ "github.com/cshabsin/conju/view/poll"
)

func main() {
	appengine.Main()
}
