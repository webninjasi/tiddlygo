package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"text/template"

	"github.com/gorilla/mux"
)

const c_configFile = "tiddlygo.json"
const c_maxFileSize = 32 << 20

type WikiList struct {
	Pages []Page `json:"pages"`
}

type Page struct {
	Url  string `json:"url"`
	Name string `json:"name"`
}

var cfg = NewConfig()
var evtHandler = EventHandler{}
var indexTpl = template.Must(template.ParseFiles("www/index.html"))

func main() {
	err := cfg.ReadFile(c_configFile)
	if err != nil && !os.IsNotExist(err) {
		log.Fatalln("Error while reading config file:", err)
	}

	evtHandler.Parse(cfg.Events)

	router := mux.NewRouter()
	router.HandleFunc("/", index).Methods("GET")
	router.HandleFunc("/wikilist", listWiki).Methods("GET")
	router.HandleFunc("/store", storeWiki).Methods("POST")
	router.HandleFunc("/new", newWiki).Methods("POST")
	router.HandleFunc("/{wikiname:\\w+\\.html}", viewWiki).Methods("GET")
	router.PathPrefix("/").Handler(http.FileServer(http.Dir("./www/")))

	log.Println("Listening the server on", cfg.Address)

	err = http.ListenAndServe(cfg.Address, router)
	if err != nil {
		log.Fatalln("Error while listening server:", err)
	}
}

func index(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "www/index.html")
}

func listWiki(w http.ResponseWriter, r *http.Request) {
	files, err := ioutil.ReadDir(cfg.WikiDir + "/")
	if err != nil {
		return
	}

	data := WikiList{
		Pages: []Page{},
	}

	for _, f := range files {
		name := f.Name()
		if len(name) > 5 && name[len(name)-5:] == ".html" {
			data.Pages = append(data.Pages, Page{
				Url:  "/" + name,
				Name: name,
			})
		}
	}

	byt, err := json.Marshal(data)
	if err != nil {
		return
	}

	w.Write(byt)
}

func viewWiki(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	wikiname := params["wikiname"]

	// Disable Caching
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")

	http.ServeFile(w, r, filepath.Join(cfg.WikiDir, wikiname))
}

func storeWiki(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(c_maxFileSize)
	if err != nil {
		log.Println("Error while parsing form:", err)
		return
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")

	optionsStr, ok := r.MultipartForm.Value["UploadPlugin"]
	if !ok || len(optionsStr) < 1 {
		fmt.Fprintf(w, "Couldn't find 'UploadPlugin' in the form data!")
		return
	}

	options := parseOptions(optionsStr[0])

	user, ok := options["user"]
	if !ok {
		fmt.Fprintf(w, "Couldn't find 'user' in the form data!")
		return
	}

	pass, ok := options["password"]
	if !ok {
		fmt.Fprintf(w, "Couldn't find 'password' in the form data!")
		return
	}

	if user != cfg.Username {
		fmt.Fprintln(w, "Error: Username do not match!")
		fmt.Fprintf(w, "Username: [%v]\n", user)
		return
	}

	if pass != cfg.Password {
		fmt.Fprintln(w, "Error: Password do not match!")
		return
	}

	uploadf, handler, err := r.FormFile("userfile")
	if err != nil {
		fmt.Fprintln(w, "Couldn't upload the file!")
		log.Println("Error while handling 'userfile':", err)
		return
	}
	defer uploadf.Close()

	wikiname := filepath.Base(handler.Filename)

	match, err := regexp.MatchString(`^\w+\.html$`, wikiname)
	if !match || err != nil {
		fmt.Fprintln(w, "Invalid file name!")
		return
	}

	wikipath := filepath.Join(cfg.WikiDir, wikiname)

	if !isExist(cfg.WikiDir) {
		err := os.MkdirAll(cfg.WikiDir, 0644)
		if err != nil {
			fmt.Fprintln(w, "Couldn't upload the file!")
			log.Printf("Error while creating '%v': %v\n", cfg.WikiDir, err)
			return
		}
	}

	downloadf, err := os.OpenFile(wikipath, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		fmt.Fprintln(w, "Couldn't upload the file!")
		log.Printf("Error while opening '%v': %v\n", wikipath, err)
		return
	}
	defer downloadf.Close()

	evtHandler.Handle("prestore", wikiname)

	_, err = io.Copy(downloadf, uploadf)
	if err != nil {
		fmt.Fprintln(w, "Couldn't upload the file!")
		log.Printf("Error while copying to '%v': %v\n", wikipath, err)
		return
	}

	evtHandler.Handle("poststore", wikiname)

	fmt.Fprintf(w, "0 - File successfully loaded in '%v'\n", wikiname)
	log.Printf("Successfully uploaded: '%v'\n", wikiname)
}

func newWiki(w http.ResponseWriter, r *http.Request) {
	wikiname := r.FormValue("wikiname")

	match, err := regexp.MatchString(`^\w+$`, wikiname)
	if !match || err != nil {
		log.Println(wikiname)
		log.Println(err)
		http.Error(w, "Invalid file name!", http.StatusBadRequest)
		return
	}

	wikiname = wikiname + ".html"
	wikipath := filepath.Join(cfg.WikiDir, wikiname)

	if isExist(wikipath) {
		http.Error(w, "It already exists!", http.StatusBadRequest)
		return
	}

	err = downloadFile(wikipath, "http://tiddlywiki.com/empty.html")
	if err != nil {
		http.Error(w, "Couldn't download an empty wiki!", http.StatusInternalServerError)
		log.Println("Error while downloading empty wiki:", err)
		return
	}

	fmt.Fprintf(w, "Success!")
}
