package main

import (
	"log"

	"github.com/NebojsaJovanovic95/VISP/internal/server"
)

func main() {
	if err := server.Run(); err != nil {
		log.Fatal(err)
	}
}
