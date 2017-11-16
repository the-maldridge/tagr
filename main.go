package main

import (
	"log"
	"fmt"
	"flag"
	"net/http"
	"html/template"
)

var (
	templates = template.Must(template.ParseFiles("tmpl/status.tmpl"))

	port = flag.Int("port", 8080, "Serving port")
)

func OKHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "OK")
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
	s := struct {
		Port int
	}{
		Port: *port,
	}
		
	
	err := templates.ExecuteTemplate(w, "status.tmpl", s)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func main() {
	log.Println("Tagr Server is initializing...")
	http.HandleFunc("/ok", OKHandler)
	http.HandleFunc("/status", statusHandler)
	http.ListenAndServe(":8080", nil)
}
