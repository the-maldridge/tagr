package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"time"
)

type LibraryEntry struct {
	Filename    string
	Title       string
	Tags        []string
	Date        time.Time
	Description string
}

type Library struct {
	Tags    []string
	Entries map[string]*LibraryEntry
}

var (
	templates = template.Must(template.ParseFiles("tmpl/status.tmpl", "tmpl/list.tmpl", "tmpl/player.tmpl"))

	port     = flag.Int("port", 8080, "Serving port")
	videoDir = flag.String("video_dir", "video", "Directory to search for files to be tagged")

	library = Library{}
)

func OKHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "OK")
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
	s := struct {
		Port     int
		VideoDir string
		Library  Library
	}{
		Port:     *port,
		VideoDir: *videoDir,
		Library:  library,
	}

	err := templates.ExecuteTemplate(w, "status.tmpl", s)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func listHandler(w http.ResponseWriter, r *http.Request) {
	err := templates.ExecuteTemplate(w, "list.tmpl", library)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func playerHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		fmt.Fprintf(w, "Error decoding request!")
	}

	err = templates.ExecuteTemplate(w, "player.tmpl", library.Entries[r.FormValue("file")].Filename)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func infoHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		log.Println("infoHandler: form parse error!")
	}

	data, err := json.Marshal(library.Entries[r.FormValue("file")])
	if err != nil {
		log.Println("infoHandler: json marshal error!")
	}

	fmt.Fprintf(w, "%s", data)
}

func findVideos() {
	log.Println("Begining video search")
	path, err := filepath.Abs(*videoDir)
	if err != nil {
		log.Fatalf("Error getting video path: %s", err)
	}
	*videoDir = path
	log.Printf("Loading videos from %s", *videoDir)

	files, err := filepath.Glob(path + "/*")
	if err != nil {
		log.Printf("Error globbing videos: %s", err)
	}

	log.Println("Located the following files:")
	for _, v := range files {
		v = filepath.Base(v)
		if library.Entries[v] == nil {
			// Add a file we haven't seen before
			log.Printf("  New File: %s", v)
			library.Entries[v] = &LibraryEntry{Filename: v}
		} else {
			log.Printf("  Known File: %s", v)
		}
	}
}

func main() {
	flag.Parse()
	log.Println("Tagr Server is initializing...")

	http.HandleFunc("/ok", OKHandler)
	http.HandleFunc("/status", statusHandler)
	http.HandleFunc("/list", listHandler)
	http.HandleFunc("/player", playerHandler)
	http.HandleFunc("/info", infoHandler)
	http.Handle("/video-file/", http.StripPrefix("/video-file/", http.FileServer(http.Dir(*videoDir))))
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))

	library.Entries = make(map[string]*LibraryEntry)

	// shim this in for testing
	library.Entries["big_buck_bunny.mp4"] = &LibraryEntry{
		Filename:    "big_buck_bunny.mp4",
		Title:       "Big Buck Bunny",
		Description: "A test film from the fine folks at Blender",
		Date: time.Now(),
		Tags: []string{"cartoon", "animal"},
	}

	findVideos()

	http.ListenAndServe(":8080", nil)
}
