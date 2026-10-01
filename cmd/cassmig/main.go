package main

import (
	"log"

	"github.com/evolvedevlab/cassmig"
)

func main() {
	if err := cassmig.Execute(); err != nil {
		log.Fatalf("cassmig error: %v", err)
	}
}
