package main

import (
	"log"

	"github.com/vas3k/pepic/pepic/cmd"
)

func main() {
	err := cmd.Execute()
	if err != nil {
		log.Fatal(err)
	}
}
