// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"time"

	skynet "github.com/NebulousLabs/go-skynet"
)

type Page struct {
	Title string
	Body  []byte
}

type SkyNote struct {
	// filename to skylink
	skynotes map[string]string

	// skylink to unix timestampe
	skylinks map[string]int64
}

var ss = &SkyNote{
	skynotes: make(map[string]string),
	skylinks: make(map[string]int64),
}

func (p *Page) save() error {
	filename := filepath.Join("pages", p.Title+".txt")
	err := ioutil.WriteFile(filename, p.Body, 0600)
	if err != nil {
		log.Println("Error writing file:", err)
		return err
	}
	// Check if current skylink renders the same content
	skylink, ok := ss.skynotes[filename]
	if ok {
		copyFile := fmt.Sprintf("%v_copy", filename)
		err = skynet.DownloadFile(copyFile, skylink, skynet.DefaultDownloadOptions)
		if err != nil {
			log.Println("Error downloading file from skynet:", err)
			return err
		}
		// Clean up downloaded file
		defer os.Remove(copyFile)

		body, err := ioutil.ReadFile(copyFile)
		if err != nil {
			log.Println("Error reading file from disk:", err)
			return err
		}
		if bytes.Equal(body, p.Body) {
			// File hasn't changed, return
			return nil
		}
	}

	// Upload new file
	skylink, err = skynet.UploadFile(filename, skynet.DefaultUploadOptions)
	if err != nil {
		log.Println("Error saving file to skynet:", err)
		return err
	}
	// Add to maps
	ss.skynotes[filename] = skylink
	ss.skylinks[skylink] = time.Now().Unix()
	fmt.Println("skylink", skylink)
	return ss.save()
}

func loadPage(title string) (*Page, error) {
	filename := filepath.Join("pages", title+".txt")
	skylink, ok := ss.skynotes[filename]
	if !ok {
		log.Println("Trying to load non existent file, this shouldn't happen")
		return nil, errors.New("note not being tracked")
	}
	err := skynet.DownloadFile(filename, skylink, skynet.DefaultDownloadOptions)
	if err != nil {
		log.Println("Error downloading file from skynet:", err)
		return nil, err
	}
	body, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return &Page{Title: title, Body: body}, nil
}

func viewHandler(w http.ResponseWriter, r *http.Request, title string) {
	p, err := loadPage(title)
	if err != nil {
		http.Redirect(w, r, "/edit/"+title, http.StatusFound)
		return
	}
	renderTemplate(w, "view", p)
}

func editHandler(w http.ResponseWriter, r *http.Request, title string) {
	p, err := loadPage(title)
	if err != nil {
		p = &Page{Title: title}
	}
	renderTemplate(w, "edit", p)
}

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

var templates = template.Must(template.ParseFiles("edit.html", "view.html"))

func renderTemplate(w http.ResponseWriter, tmpl string, p *Page) {
	err := templates.ExecuteTemplate(w, tmpl+".html", p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

var validPath = regexp.MustCompile("^/(edit|save|view)/([a-zA-Z0-9]+)$")

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
	err := ss.load()
	if err != nil {
		panic(err)
	}
	http.HandleFunc("/view/", makeHandler(viewHandler))
	http.HandleFunc("/edit/", makeHandler(editHandler))
	http.HandleFunc("/save/", makeHandler(saveHandler))

	log.Fatal(http.ListenAndServe(":8080", nil))
}
