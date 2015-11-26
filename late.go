package main

import (
	"os"
	"log"
	"io/ioutil"
	"path/filepath"
	"html/template"
	"strconv"
	"net/http"
)

// directory struct: {root}/{book title}/{chapter num}/{block num}
// when program begin, it will look through the dirs
// and make following structs.

type book struct {
	Title string
	Chapters []chapter
}

type chapter struct {
	Num int
	Snippets []snippet
}

// snippet is a string block of resonable length.
// user will decide how long will it be.
type snippet struct {
	Orig string
	Trans string
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	t, err := template.ParseFiles("index.html")
	if err != nil {
		log.Fatal(err)
	}
	books := scanRootDir()
	err = t.Execute(w, books[0])
	if err != nil {
		log.Fatal(err)
	}
}

func updateHandler(w http.ResponseWriter, r *http.Request, rootpath string) {
	r.ParseForm()
	// Why r.Form["x"] get in form with []string ?
	typ := r.Form["type"][0]
	subpath := r.Form["path"][0]
	orig := r.Form["orig"][0]
	trans := r.Form["trans"][0]
	path := filepath.Join(rootpath, subpath)
	switch typ {
	case "save":
		saveSnippet(path, orig, trans)
	case "new":
		newSnippet(path, orig, trans)
	}
}

func saveSnippet(path, orig, trans string) {
	err := ioutil.WriteFile(path + "/orig", []byte(orig), 0644)
	if ( err != nil ) {
		log.Fatal(err)
	}
	err = ioutil.WriteFile(path + "/trans", []byte(trans), 0644)
	if ( err != nil ) {
		log.Fatal(err)
	}
}

func newSnippet(path, orig, trans string) {
	err := os.MkdirAll(path, 0755)
	if ( err != nil ) {
		log.Fatal(err)
	}
	saveSnippet(path, orig, trans)
}

// scanRootDir scan and return books in root dir with alphabetical order.
func scanRootDir() []book {
	root := "./testroot"
	bookFiles, err := ioutil.ReadDir(root)
	if err != nil {
		log.Fatal(err)
	}
	books := make([]book, 0)
	for _, bf := range bookFiles {
		if !bf.IsDir() {
			continue
		}
		bk := book{Title:bf.Name(), Chapters:make([]chapter, 0)}
		bdir := filepath.Join(root, bf.Name())
		chapterFiles, err := ioutil.ReadDir(bdir)
		if err != nil {
			log.Fatal(err)
		}
		for _, cf := range chapterFiles {
			if !cf.IsDir() {
				continue
			}
			cnum, err := strconv.Atoi(cf.Name())
			if err != nil {
				log.Fatal(err)
			}
			chap := chapter{Num:cnum, Snippets:make([]snippet, 0)}
			cdir := filepath.Join(bdir, cf.Name())
			snippetFiles, err := ioutil.ReadDir(cdir)
			if err != nil {
				log.Fatal(err)
			}
			var orig, trans string
			for _, f := range snippetFiles {
				if !f.IsDir() {
					continue
				}
				sdir := filepath.Join(cdir, f.Name())
				files, err := ioutil.ReadDir(sdir)
				if err != nil {
					log.Fatal(err)
				}
				for _, f := range files {
					// lasts are files
					if f.IsDir() {
						continue
					}
					fpath := filepath.Join(sdir, f.Name())
					if f.Name() == "orig" {
						o, err := ioutil.ReadFile(fpath)
						if err != nil {
							log.Fatal(err)
						}
						orig = string(o)
					} else if f.Name() == "trans" {
						t, err := ioutil.ReadFile(fpath)
						if err != nil {
							log.Fatal(err)
						}
						trans = string(t)
					}
				}
				snip := snippet{Orig:orig, Trans:trans}
				chap.Snippets = append(chap.Snippets, snip)
			}
			bk.Chapters = append(bk.Chapters, chap)
		}
		books = append(books, bk)
	}
	return books
}

func main() {
	rootpath := "testroot"

	http.HandleFunc("/update", func(w http.ResponseWriter, r *http.Request) {
		updateHandler(w, r, rootpath)
	})

	fs := http.StripPrefix("/script/", http.FileServer(http.Dir("script/")))
	http.Handle("/script/", fs)

	fs = http.StripPrefix("/css/", http.FileServer(http.Dir("css/")))
	http.Handle("/css/", fs)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		rootHandler(w, r)
	})
	http.ListenAndServe(":8080", nil)
}
