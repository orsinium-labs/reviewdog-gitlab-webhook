package main

import (
	"log"
	"net/http"
)

func main() {
	s, err := NewServer()
	if err != nil {
		log.Fatalf("cannot create server: %v", err)
	}
	http.HandleFunc("/review", s.Handle)
	s.logger.InfoWith("listening").String("address", s.Address).Write()
	log.Fatal(http.ListenAndServe(s.Address, nil))
}
