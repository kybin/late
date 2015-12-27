package main

import (
	"os"
	"log"
	"io/ioutil"
	"path/filepath"
	"html/template"
	"strconv"
	"net/http"
	"sort"
	"fmt"
	"strings"
	"bytes"
	"encoding/json"
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
	Orig string `json:"orig"`
	Trans string `json:"trans"`
}

// it grouping books by 4.
func grouping(books []book) [][]book {
	grps := make([][]book, 0)
	g := make([]book, 0)
	for _, b := range books {
		g = append(g, b)
		if len(g) >= 4 {
			grps = append(grps, g)
			g = make([]book, 0)
		}
	}
	if len(g) > 0 {
		grps = append(grps, g)
	}
	return grps
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	books := scanRootDir()
	b := struct{
		BookGroups [][]book
	}{
		BookGroups: grouping(books),
	}
	err := indexTemplate.Execute(w, b)
	if err != nil {
		log.Fatal(err)
	}
}

func docHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")

	log.Println(r.URL.Path)
	paths := strings.Split(r.URL.Path, "/")

	title := paths[len(paths)-2]
	chapNumStr := paths[len(paths)-1]
	chapNum, err := strconv.Atoi(chapNumStr)
	if err != nil {
		log.Println(err)
		fmt.Fprintf(w, "not found the page")
		return
	}

	books := scanRootDir()

	bi := -1
	for i, b := range books {
		if b.Title == title {
			bi = i
			break
		}
	}
	if bi == -1 {
		fmt.Fprintf(w, "not found the page")
		return
	}
	bk := books[bi]

	ci := -1
	for i, c := range bk.Chapters {
		if c.Num == chapNum {
			ci = i
		}
	}
	if ci == -1 {
		fmt.Fprintf(w, "not found the page")
		return
	}
	chap := bk.Chapters[ci]

	data := struct{
		Book book
		Chapter chapter
	}{
		Book: bk,
		Chapter: chap,
	}

	err = docTemplate.Execute(w, data)
	if err != nil {
		log.Fatal(err)
	}
}

func newDocumentHandler(w http.ResponseWriter, r *http.Request, rootpath string) {
	r.ParseForm()
	title := r.Form["title"][0]
	// also create the first chapter
	path := filepath.Join(rootpath, title, "0")
	err := os.MkdirAll(path, 0755)
	if err != nil {
		log.Fatal(err)
	}
}

func removeDocumentHandler(w http.ResponseWriter, r *http.Request, rootpath string) {
	r.ParseForm()
	subpath := r.Form["path"][0]
	path := filepath.Join(rootpath, subpath)
	removeDocument(path)
}

func removeDocument(path string) {
	if path == "" {
		log.Fatal("should not delete late root")
	}
	err := os.RemoveAll(path)
	if err != nil {
		log.Fatal(err)
	}
}

func newChapterHandler(w http.ResponseWriter, r *http.Request, rootpath string) {
	log.Println("HI")
	r.ParseForm()
	subpath := r.Form["path"][0]
	path := filepath.Join(rootpath, subpath)
	log.Println(path)
	newChapter(path)
}

func newChapter(path string) {
	err := os.MkdirAll(path, 0755)
	if err != nil {
		log.Fatal(err)
	}
}

func saveSnippetHandler(w http.ResponseWriter, r *http.Request, rootpath string) {
	r.ParseForm()
	subpath := r.Form["path"][0]
	path := filepath.Join(rootpath, subpath)
	orig := r.Form["orig"][0]
	trans := r.Form["trans"][0]
	saveSnippet(path, orig, trans)
}

func saveSnippet(path, orig, trans string) {
	err := ioutil.WriteFile(path + "/orig", []byte(orig), 0644)
	if err != nil {
		log.Fatal(err)
	}
	err = ioutil.WriteFile(path + "/trans", []byte(trans), 0644)
	if err != nil {
		log.Fatal(err)
	}
}

// contentsSnippetHandler handles /contents/snippet request,
// it responses original contents of orig, trans snippets as json format.
func contentsSnippetHandler(w http.ResponseWriter, r *http.Request, rootpath string) {
	r.ParseForm()
	subpath := r.Form["path"][0]
	path := filepath.Join(rootpath, subpath)

	orig, err := ioutil.ReadFile(path + "/orig")
	if err != nil {
		log.Fatal(err)
	}
	trans, err := ioutil.ReadFile(path + "/trans")
	if err != nil {
		log.Fatal(err)
	}

	snip := snippet{
		Orig: string(orig),
		Trans: string(trans),
	}

	js, err := json.Marshal(snip)
	if err != nil {
		log.Fatal(err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

func insertSnippetHandler(w http.ResponseWriter, r *http.Request, rootpath string) {
	r.ParseForm()
	subpath := r.Form["path"][0]
	path := filepath.Join(rootpath, subpath)
	insertSnippet(path)
}

func insertSnippet(path string) {
	// before insert the snippet dir, the later dirs should shifted by 1.
	// if 2 is inserted. n -> n + 1, .. 4 -> 3, 2 -> 3.
	chapd, snipd := filepath.Split(path)
	insertIndex, err := strconv.Atoi(snipd)
	if err != nil {
		log.Fatal(err)
	}
	files, err := ioutil.ReadDir(chapd)
	if err != nil {
		log.Fatal(err)
	}
	sort.Sort(sort.Reverse(byIndex(files)))
	for _, f := range files {
		if !f.IsDir() {
			log.Fatal("chapter directory should only have snippet directories. file found")
		}
		i, err := strconv.Atoi(f.Name())
		if err != nil {
			log.Fatal(err)
		}
		if i < insertIndex {
			break
		}
		newName := strconv.Itoa(i+1)
		err = os.Rename(filepath.Join(chapd, f.Name()), filepath.Join(chapd, newName))
		if err != nil {
			log.Fatal(err)
		}
	}

	err = os.MkdirAll(path, 0755)
	if err != nil {
		log.Fatal(err)
	}
	err = ioutil.WriteFile(path + "/orig", []byte(""), 0644)
	if err != nil {
		log.Fatal(err)
	}
	err = ioutil.WriteFile(path + "/trans", []byte(""), 0644)
	if err != nil {
		log.Fatal(err)
	}
}

func removeSnippetHandler(w http.ResponseWriter, r *http.Request, rootpath string) {
	r.ParseForm()
	subpath := r.Form["path"][0]
	path := filepath.Join(rootpath, subpath)
	removeSnippet(path)
}

func removeSnippet(path string) {
	if path == "" {
		log.Fatal("should not delete late root")
	}
	err := os.Remove(filepath.Join(path, "orig"));
	if err != nil {
		log.Fatal(err)
	}
	err = os.Remove(filepath.Join(path, "trans"));
	if err != nil {
		log.Fatal(err)
	}
	err = os.Remove(path)
	if err != nil {
		log.Fatal(err)
	}
	// after remove the snippet dir, the later dirs should renamed to fill hole.
	// if 2 is removed. 3 -> 2, 4 -> 3, .. n + 1 -> n
	chapd, snipd := filepath.Split(path)
	rmIndex, err := strconv.Atoi(snipd)
	if err != nil {
		log.Fatal(err)
	}
	files, err := ioutil.ReadDir(chapd)
	if err != nil {
		log.Fatal(err)
	}
	sort.Sort(byIndex(files))
	for _, f := range files {
		if !f.IsDir() {
			log.Fatal("chapter directory should only have snippet directories. file found")
		}
		i, err := strconv.Atoi(f.Name())
		if err != nil {
			log.Fatal(err)
		}
		if i <= rmIndex {
			continue
		}
		newName := strconv.Itoa(i-1)
		err = os.Rename(filepath.Join(chapd, f.Name()), filepath.Join(chapd, newName))
		if err != nil {
			log.Fatal(err)
		}
	}
}

// onelineSnippetHander handles request to /oneline/snippet.
// When data copied from other documents (eg. pdf). they often have line endings for themselves.
// So sometimes user wants remove windows/unix line endings from orig/trans snippet files.
// It replaces each line ending to a space.
func onelineSnippetHander(w http.ResponseWriter, r *http.Request, rootpath string) {
	r.ParseForm()
	subpath := r.Form["path"][0]
	path := filepath.Join(rootpath, subpath)
	log.Println("oneline : " + path)

	orig, err := ioutil.ReadFile(path+"/orig")
	if err != nil {
		log.Fatal(err)
	}
	orig = bytes.Replace(orig, []byte("\r\n"), []byte(" "), -1)
	orig = bytes.Replace(orig, []byte("\n"), []byte(" "), -1)
	err = ioutil.WriteFile(path+"/orig", orig, 0644)
	if err != nil {
		log.Fatal(err)
	}

	trans, err := ioutil.ReadFile(path+"/trans")
	if err != nil {
		log.Fatal(err)
	}
	trans = bytes.Replace(trans, []byte("\r\n"), []byte(" "), -1)
	trans = bytes.Replace(trans, []byte("\n"), []byte(" "), -1)
	err = ioutil.WriteFile(path+"/trans", trans, 0644)
	if err != nil {
		log.Fatal(err)
	}

	snip := snippet{
		Orig: string(orig),
		Trans: string(trans),
	}

	js, err := json.Marshal(snip)
	if err != nil {
		log.Fatal(err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

type byIndex []os.FileInfo

func (b byIndex) Len() int {
	return len(b)
}

func (b byIndex) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}

func (b byIndex) Less(i, j int) bool {
	return atoi(b[i].Name()) < atoi(b[j].Name())
}

func atoi(s string) int {
	i, err := strconv.Atoi(s)
	if err != nil {
		log.Fatal(err)
	}
	return i
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
		sort.Sort(byIndex(chapterFiles))
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
			sort.Sort(byIndex(snippetFiles))
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

var (
	indexTemplate *template.Template
	docTemplate *template.Template
)

func init() {
	indexTemplate = template.Must(template.ParseFiles("index.html"))
	docTemplate = template.Must(template.ParseFiles("doc.html"))
}

func main() {
	rootpath := "testroot"

	http.Handle("/script/", http.StripPrefix("/script/", http.FileServer(http.Dir("script/"))))
	http.Handle("/css/", http.StripPrefix("/css/", http.FileServer(http.Dir("css/"))))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		rootHandler(w, r)
	})
	http.HandleFunc("/doc/", func(w http.ResponseWriter, r *http.Request) {
		docHandler(w, r)
	})
	http.HandleFunc("/new/doc", func(w http.ResponseWriter, r *http.Request) {
		newDocumentHandler(w, r, rootpath)
	})
	http.HandleFunc("/remove/doc", func(w http.ResponseWriter, r *http.Request) {
		removeDocumentHandler(w, r, rootpath)
	})
	http.HandleFunc("/new/chapter", func(w http.ResponseWriter, r *http.Request) {
		newChapterHandler(w, r, rootpath)
	})
	http.HandleFunc("/save/snippet", func(w http.ResponseWriter, r *http.Request) {
		saveSnippetHandler(w, r, rootpath)
	})
	http.HandleFunc("/contents/snippet", func(w http.ResponseWriter, r *http.Request) {
		contentsSnippetHandler(w, r, rootpath)
	})
	http.HandleFunc("/insert/snippet", func(w http.ResponseWriter, r *http.Request) {
		insertSnippetHandler(w, r, rootpath)
	})
	http.HandleFunc("/remove/snippet", func(w http.ResponseWriter, r *http.Request) {
		removeSnippetHandler(w, r, rootpath)
	})
	http.HandleFunc("/oneline/snippet", func(w http.ResponseWriter, r *http.Request) {
		onelineSnippetHander(w, r, rootpath)
	})

	err := http.ListenAndServe(":8080", nil)
	if ( err != nil ) {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
