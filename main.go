package main

import (
	"log"

	"fizhub/cmd/fizhub"
)

func main() {
	log.Println("Starting FizHub application...")
	if err := fizhub.Run(); err != nil {
		log.Fatalf("Application error: %v", err)
	}
}
