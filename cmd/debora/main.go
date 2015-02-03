package main

import (
	"github.com/ebuchman/debora"
	"log"
)

func main() {
	err := debora.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}
}
