package main

import (
	"log"

	"github.com/fcjr/geticon"
)

func main() {
	_, err := geticon.FromPid(4656)
	if err != nil {
		log.Fatalf(err.Error())
	}
}
