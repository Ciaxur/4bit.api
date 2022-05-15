package main

import (
	"log"

	"4bit.api/v0/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
