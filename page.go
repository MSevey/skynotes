package main

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"time"

	skynet "github.com/NebulousLabs/go-skynet"
)

// Page contains the information that is rendered on the webpage
type Page struct {
	Title string
	Body  []byte
	Notes []string
}

// save saves the Page
func (p *Page) save() error {
	if len(p.Body) == 0 {
		return nil
	}
	filename := filepath.Join(skyNotePersistDir(), p.Title)
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
	return ss.save()
}

// loadPage loads a page from a skylink
func loadPage(title string) (*Page, error) {
	filename := filepath.Join(skyNotePersistDir(), title)
	skylink, ok := ss.skynotes[filename]
	if !ok {
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
