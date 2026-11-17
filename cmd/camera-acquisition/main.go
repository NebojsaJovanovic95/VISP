package main

import (
	"log"

	"github.com/NebojsaJovanovic95/VISP/internal/acquisition"
)

func main() {
	if err := acquisition.Run(); err != nil {
		log.Fatal(err)
	}
}
