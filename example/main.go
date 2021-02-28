package main

import (
	"image/png"
	"log"
	"os"
	"strconv"

	"github.com/fcjr/geticon"
)

func main() {

	arg := os.Args[1]

	pid, err := strconv.Atoi(arg)
	if err != nil {
		log.Fatal(err)
	}

	img, err := geticon.FromPid(uint32(pid))
	if err != nil {
		log.Fatal(err)
	}

	outputFile, err := os.Create("image.png")
	if err != nil {
		log.Fatal(err)
	}
	defer outputFile.Close()

	err = png.Encode(outputFile, img)
	if err != nil {
		log.Fatal(err)
	}
}
