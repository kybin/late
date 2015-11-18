package main

import (
	"fmt"
	"log"
	"io/ioutil"
	"path/filepath"
)

// directory struct: {root}/{book title}/{chapter num}/{block num}
// when program begin, it will look through the dirs
// and make following structs.

type book struct {
	title string
	chapters []chapter
}

type chapter struct {
	snippets []snippet
}

// snippet is a string block of resonable length.
// user will decide how long will it be.
type snippet struct {
	orig string
	trans string
}

func main() {
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
		bk := book{title:bf.Name(), chapters:make([]chapter, 0)}
		bdir := filepath.Join(root, bf.Name())
		chapterFiles, err := ioutil.ReadDir(bdir)
		if err != nil {
			log.Fatal(err)
		}
		for _, cf := range chapterFiles {
			if !cf.IsDir() {
				continue
			}
			chap := chapter{snippets:make([]snippet, 0)}
			cdir := filepath.Join(bdir, cf.Name())
			snippetFiles, err := ioutil.ReadDir(cdir)
			if err != nil {
				log.Fatal(err)
			}
			var orig, trans string
			for _, f := range snippetFiles {
				// lasts are files
				if f.IsDir() {
					continue
				}
				fpath := filepath.Join(cdir, f.Name())
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
			snip := snippet{orig:orig, trans:trans}
			chap.snippets = append(chap.snippets, snip)
			bk.chapters = append(bk.chapters, chap)
		}
		books = append(books, bk)
	}
	fmt.Println(books)
}
