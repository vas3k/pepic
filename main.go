package main

import (
	"github.com/vas3k/pepic/pepic/cmd"
	"log"
)

func main() {
	err := cmd.Execute()
	if err != nil {
		log.Fatal(err)
	}
}
