package main

import (
	"flag"
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
const c_version = "1.0.0"

type IndexData struct {
	Pages   []Page
	Version string
}

type Page struct {
	Url  string
	Name string
}

var cfg = NewConfig()
var indexTpl = template.Must(template.ParseFiles("www/index.html"))

var flagVersion = flag.Bool("v", false, "Show current version")

func main() {
	if *flagVersion {
		fmt.Println("TiddlyGo v" + c_version)
		return
	}

	err := cfg.ReadFile(c_configFile)
	if err != nil && !os.IsNotExist(err) {
		log.Fatalln("Error while reading config file:", err)
	}

	router := mux.NewRouter()
	router.HandleFunc("/", index).Methods("GET")
	router.HandleFunc("/store", storeWiki).Methods("POST")
	router.HandleFunc("/{wikiname:\\w+\\.html}", viewWiki).Methods("GET")
	router.PathPrefix("/").Handler(http.FileServer(http.Dir("./www/")))

	log.Println("Listening the server on", cfg.Address)

	err = http.ListenAndServe(cfg.Address, router)
	if err != nil {
		log.Fatalln("Error while listening server:", err)
	}
}

func index(w http.ResponseWriter, r *http.Request) {
	files, err := ioutil.ReadDir(cfg.WikiDir + "/")
	if err != nil {
		return
	}

	data := IndexData{
		Pages:   []Page{},
		Version: c_version,
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

	err = indexTpl.Execute(w, data)
	if err != nil {
		fmt.Fprintf(w, `Template error`)
		log.Println(err)
		return
	}
}

func viewWiki(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	wikiname := params["wikiname"]
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

	_, err = io.Copy(downloadf, uploadf)
	if err != nil {
		fmt.Fprintln(w, "Couldn't upload the file!")
		log.Printf("Error while copying to '%v': %v\n", wikipath, err)
		return
	}

	fmt.Fprintf(w, "0 - File successfully loaded in '%v'\n", wikiname)
	log.Printf("Successfully uploaded: '%v'\n", wikiname)
}
