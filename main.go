package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io/ioutil"
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

	port         = flag.Int("port", 8080, "Serving port")
	videoDir     = flag.String("video_dir", "video", "Directory to search for files to be tagged")
	saveInterval = flag.Duration("save_interval", 5*time.Minute, "How often to back up the database to disk")

	library = Library{}
	healthy = "OK"
	dbDirty = false
)

func OKHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, healthy)
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

	json.NewEncoder(w).Encode(library.Entries[r.FormValue("file")])
}

func updateHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		log.Println("updateHandler: form parse error!")
	}

	file := r.FormValue("file")
	if file == "" {
		log.Println("malformed update request!")
		return
	}

	// The update process is a complete overwrite every time it is
	// run, for this reason we have to make sure the form on the
	// viewer page is complete before sending it back.  This needs
	// to be communicated to the user as a manual expectation.
	entry := &LibraryEntry{}
	err = json.NewDecoder(r.Body).Decode(&entry)
	if err != nil {
		log.Println("updateHandler: json decode fault!")
	}
	log.Printf("Updating metadata for %s", file)
	library.Entries[file] = entry

	// mark the DB dirty, this causes the backup to actually do things
	dbDirty = true
}

func dbDumpHandler(w http.ResponseWriter, r *http.Request) {
	// This function is almost entirely for dumping the database
	// during test or similar purposes.
	json.NewEncoder(w).Encode(library)
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

func dbBackup() {
	log.Println("Backup up database")
	d, err := json.Marshal(library)
	if err != nil {
		log.Println("Marshaling error during database backup!")
		// flip the global status to bad here
		healthy = "NOT OK"
	}
	err = ioutil.WriteFile("tagr.json", d, 0644)
	if err != nil {
		log.Println("File Write error during database backup!")
		healthy = "NOT OK"
	}
	log.Println("Database backup complete")
}

func dbBackupTimer() {
	for range time.Tick(*saveInterval) {
		if dbDirty {
			dbBackup()
			dbDirty = false
		}
	}
}

func dbLoad() {
	d, err := ioutil.ReadFile("tagr.json")
	if err != nil {
		log.Fatalf("Could not load database: %s", err)
	}
	err = json.Unmarshal(d, &library)
	if err != nil {
		log.Fatalf("Could not unpack database: %s", err)
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
	http.HandleFunc("/update", updateHandler)
	http.HandleFunc("/db", dbDumpHandler)
	http.Handle("/video-file/", http.StripPrefix("/video-file/", http.FileServer(http.Dir(*videoDir))))
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))

	library.Entries = make(map[string]*LibraryEntry)

	// shim this in for testing
	library.Entries["big_buck_bunny.mp4"] = &LibraryEntry{
		Filename:    "big_buck_bunny.mp4",
		Title:       "Big Buck Bunny",
		Description: "A test film from the fine folks at Blender",
		Date:        time.Now(),
		Tags:        []string{"cartoon", "animal"},
	}

	dbLoad()
	findVideos()
	dbBackup()

	// launch the backup goroutine
	go dbBackupTimer()

	http.ListenAndServe(":8080", nil)
}
