package main

import (
	"os"
	"path/filepath"
	"runtime"

	"gitlab.com/NebulousLabs/Sia/persist"
)

var (
	// persistMetadata is the persistence metadata for the SkyNotes
	persistMetadata = persist.Metadata{
		Header:  "SkyNotes Persistence",
		Version: "v0.1.0",
	}

	// persistFileName is the filename for the persistence file on disk
	persistFileName = filepath.Join(skyNotePersistDir(), "skynotes.json")
)

type (
	// persistedFile is the persisted file information that is stored on disk
	persistedFile struct {
		SkyLink  string `json:"skylink"`
		Filename string `json:"filename"`
	}
	persistedLink struct {
		Timestamp int64  `json:"timestamp"`
		SkyLink   string `json:"skylink"`
	}

	// persistence is the data that is stored on disk for the SkyNotes
	persistence struct {
		Files []persistedFile `json:"files"`
		Links []persistedLink `json:"links"`
	}
)

// skyNotePersistDir returns the directory that the skynote persistence will be
// saved
func skyNotePersistDir() string {
	switch runtime.GOOS {
	case "windows":
		return filepath.Join(os.Getenv("LOCALAPPDATA"), "SkyNote")
	case "darwin":
		return filepath.Join(os.Getenv("HOME"), "Library", "Application Support", "SkyNote")
	default:
		return filepath.Join(os.Getenv("HOME"), ".skynote")
	}
}

// load loads the SkyNotes's persistence from disk
func (sn *SkyNote) load() error {
	var data persistence
	err := persist.LoadJSON(persistMetadata, &data, persistFileName)
	if os.IsNotExist(err) {
		err := os.MkdirAll(skyNotePersistDir(), 0700)
		if err != nil {
			return err
		}
		err = sn.save()
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	for _, file := range data.Files {
		sn.skynotes[file.Filename] = file.SkyLink
	}
	for _, link := range data.Links {
		sn.skylinks[link.SkyLink] = link.Timestamp
	}
	return nil
}

// persistData returns the data to be stored on disk in the persistence format
func (sn *SkyNote) persistData() persistence {
	var data persistence
	for file, skylink := range sn.skynotes {
		data.Files = append(data.Files, persistedFile{
			Filename: file,
			SkyLink:  skylink,
		})
	}
	for skylink, timestamp := range sn.skylinks {
		data.Links = append(data.Links, persistedLink{
			Timestamp: timestamp,
			SkyLink:   skylink,
		})
	}
	return data
}

// save saves the SkyNote's persistence to disk
func (sn *SkyNote) save() error {
	return persist.SaveJSON(persistMetadata, sn.persistData(), persistFileName)
}
