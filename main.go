package main

import (
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
)

var (
	templates = template.Must(template.ParseFiles("tmpl/status.tmpl", "tmpl/list.tmpl", "tmpl/player.tmpl"))

	port     = flag.Int("port", 8080, "Serving port")
	videoDir = flag.String("video_dir", "video", "Directory to search for files to be tagged")

	videos []string
)

func OKHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "OK")
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
	s := struct {
		Port     int
		VideoDir string
		Videos   []string
	}{
		Port:     *port,
		VideoDir: *videoDir,
		Videos:   videos,
	}

	err := templates.ExecuteTemplate(w, "status.tmpl", s)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func listHandler(w http.ResponseWriter, r *http.Request) {
	s := struct {
		Videos   []string
	}{
		Videos:   videos,
	}

	err := templates.ExecuteTemplate(w, "list.tmpl", s)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	
}

func playerHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		fmt.Fprintf(w, "Error decoding request!")
	}

	s := struct {
		File string
	}{
		File: r.FormValue("file"),
	}
	
	err = templates.ExecuteTemplate(w, "player.tmpl", s)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func findVideos() {
	log.Println("Begining video search")
	path, err := filepath.Abs(*videoDir)
	if err != nil {
		log.Fatalf("Error getting video path: %s", err)
	}
	*videoDir = path
	log.Printf("Loading videos from %s", *videoDir)

	videos, err = filepath.Glob(path + "/*")
	if err != nil {
		log.Printf("Error globbing videos: %s", err)
	}

	log.Println("Located the following files:")
	for i, v := range videos {
		v = filepath.Base(v)
		videos[i] = v
		log.Printf("  %s", v)
	}
}

func main() {
	flag.Parse()
	log.Println("Tagr Server is initializing...")

	http.HandleFunc("/ok", OKHandler)
	http.HandleFunc("/status", statusHandler)
	http.HandleFunc("/list", listHandler)
	http.HandleFunc("/player", playerHandler)
	http.Handle("/video-file/", http.StripPrefix("/video-file/", http.FileServer(http.Dir(*videoDir))))
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))

	findVideos()

	http.ListenAndServe(":8080", nil)
}
