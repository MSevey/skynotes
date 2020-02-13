package main

import (
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"regexp"
)

// SkyNote tracks the information related to the notes stored on skynet
type SkyNote struct {
	// filename to skylink
	skynotes map[string]string

	// skylink to unix timestamp
	skylinks map[string]int64
}

var (
	// Initialize SkyNote
	ss = &SkyNote{
		skynotes: make(map[string]string),
		skylinks: make(map[string]int64),
	}

	// List of accepted templates
	templates = template.Must(template.ParseFiles("edit.html", "view.html", "skynotes.html"))

	// validPath validates the url paths
	validPath = regexp.MustCompile("^/(edit|save|view)/([a-zA-Z0-9]+)$")
)

// editHandler handles the events for /edit
func editHandler(w http.ResponseWriter, r *http.Request, title string) {
	p, err := loadPage(title)
	if err != nil {
		p = &Page{Title: title}
	}
	renderTemplate(w, "edit", p)
}

// saveHandler handles the events for /save
func saveHandler(w http.ResponseWriter, r *http.Request, title string) {
	body := r.FormValue("body")
	p := &Page{Title: title, Body: []byte(body)}
	err := p.save()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/view/"+title, http.StatusFound)
}

// skynoteHandler handles the events for /
func skynotesHandler(w http.ResponseWriter, r *http.Request) {
	// Load current notes
	var notes []string
	for note := range ss.skynotes {
		title := filepath.Base(note)
		notes = append(notes, title)
	}
	renderTemplate(w, "skynotes", &Page{Notes: notes})
}

// viewHandler handles the events for /view
func viewHandler(w http.ResponseWriter, r *http.Request, title string) {
	p, err := loadPage(title)
	if err != nil {
		http.Redirect(w, r, "/edit/"+title, http.StatusFound)
		return
	}
	renderTemplate(w, "view", p)
}

// renderTemplate renders the template for the provided tmpl string
func renderTemplate(w http.ResponseWriter, tmpl string, p *Page) {
	err := templates.ExecuteTemplate(w, tmpl+".html", p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// makeHandler makes the handler functions
func makeHandler(fn func(http.ResponseWriter, *http.Request, string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		m := validPath.FindStringSubmatch(r.URL.Path)
		if m == nil {
			http.NotFound(w, r)
			return
		}
		fn(w, r, m[2])
	}
}

func main() {
	// Try and load any persisted state for the SkyNotes
	err := ss.load()
	if err != nil {
		panic(err)
	}

	// Initialize the handles
	http.HandleFunc("/", skynotesHandler)
	http.HandleFunc("/view/", makeHandler(viewHandler))
	http.HandleFunc("/edit/", makeHandler(editHandler))
	http.HandleFunc("/save/", makeHandler(saveHandler))

	// Launch the Server
	log.Fatal(http.ListenAndServe(":8080", nil))
}
