package main

import (
	"log"

	"4bit.api/v0/cmd"
)

var (
	// Set via build flags: ie. go build -ldflags "-X 'cmd.version=1.2.3'"
	version = "dev"
)

func main() {
	if err := cmd.Execute(version); err != nil {
		log.Fatal(err)
	}
}
