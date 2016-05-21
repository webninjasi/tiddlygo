package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/cratonica/trayhost"
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

type WikiTemplate struct {
	Id       string `json:"id"`
	Name     string `json:"name"`
	Selected bool   `json:"selected"`
}

var cfg = NewConfig()
var evtHandler = EventHandler{}
var serverURL string

func main() {
	// EnterLoop must be called on the OS's main thread
	runtime.LockOSThread()

	err := cfg.ReadFile(c_configFile)
	if err != nil && !os.IsNotExist(err) {
		log.Fatalln("Error while reading config file:", err)
	}

	evtHandler.Parse(cfg.Events)

	router := getRouter()

	go func() {
		log.Println("Listening the server on", cfg.Address)

		err := http.ListenAndServe(cfg.Address, router)
		if err != nil {
			log.Fatalln("Error while listening server:", err)
		}
	}()

	serverURL = toHttpAddr(cfg.Address)

	trayhost.SetUrl(serverURL)
	trayhost.EnterLoop("TiddlyGo", iconData)
}

func getRouter() *mux.Router {
	router := mux.NewRouter()
	router.HandleFunc("/", index).Methods("GET")
	router.HandleFunc("/wikilist", listWiki).Methods("GET")
	router.HandleFunc("/wikitemplates", listWikiTemplates).Methods("GET")
	router.HandleFunc("/store", storeWiki).Methods("POST")
	router.HandleFunc("/new", newWiki).Methods("POST")
	router.HandleFunc("/{wikiname:\\w+\\.html}", viewWiki).Methods("GET")
	router.PathPrefix("/").Handler(http.FileServer(http.Dir(cfg.PublicDir)))

	return router
}

func index(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, cfg.PublicDir+"/index.html")
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

func listWikiTemplates(w http.ResponseWriter, r *http.Request) {
	files, err := ioutil.ReadDir(cfg.TemplateDir)
	if err != nil {
		return
	}

	data := []WikiTemplate{
		WikiTemplate{
			Id:   "Latest",
			Name: "Latest",
		},
	}

	for i, f := range files {
		name := f.Name()
		if len(name) > 5 && name[len(name)-5:] == ".html" && name != "Latest" {
			data = append(data, WikiTemplate{
				Id:       name,
				Name:     name,
				Selected: i == 0,
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

	inp, handler, err := r.FormFile("userfile")
	if err != nil {
		fmt.Fprintln(w, "Couldn't upload the file!")
		log.Println("Error while handling 'userfile':", err)
		return
	}
	defer inp.Close()

	wikiname := filepath.Base(handler.Filename)

	match, err := regexp.MatchString(`^\w+\.html$`, wikiname)
	if !match || err != nil {
		fmt.Fprintln(w, "Invalid file name!")
		return
	}

	wikipath := filepath.Join(cfg.WikiDir, wikiname)

	err = checkWikiDir()
	if err != nil {
		fmt.Fprintln(w, "Couldn't upload the file!")
		log.Printf("Error while creating '%v': %v\n", cfg.WikiDir, err)
		return
	}

	out, err := os.Create(wikipath)
	if err != nil {
		fmt.Fprintln(w, "Couldn't upload the file!")
		log.Printf("Error while creating '%v': %v\n", wikipath, err)
		return
	}
	defer out.Close()

	evtHandler.Handle("prestore", wikiname)

	_, err = io.Copy(out, inp)
	if err != nil {
		fmt.Fprintln(w, "Couldn't upload the file!")
		log.Printf("Error while copying to '%v': %v\n", wikipath, err)
		return
	}

	err = out.Sync()
	if err != nil {
		fmt.Fprintln(w, "Couldn't upload the file!")
		log.Printf("Error while syncing '%v': %v\n", wikipath, err)
		return
	}

	evtHandler.Handle("poststore", wikiname)

	fmt.Fprintf(w, "0 - File successfully loaded in '%v'\n", wikiname)
	log.Printf("Successfully uploaded: '%v'\n", wikiname)
}

func newWiki(w http.ResponseWriter, r *http.Request) {
	wikiname := r.FormValue("wikiname")
	wikitemplate := r.FormValue("wikitemplate")

	match, err := regexp.MatchString(`^\w+$`, wikiname)
	if !match || err != nil {
		http.Error(w, "Invalid file name!", http.StatusBadRequest)
		return
	}

	wikiname = wikiname + ".html"
	wikipath := filepath.Join(cfg.WikiDir, wikiname)

	if isExist(wikipath) {
		http.Error(w, "It already exists!", http.StatusBadRequest)
		return
	}

	err = checkWikiDir()
	if err != nil {
		fmt.Fprintln(w, "Couldn't create the wiki!")
		log.Printf("Error while creating '%v': %v\n", cfg.WikiDir, err)
		return
	}

	if wikitemplate == "Latest" {
		err = downloadFile(wikipath, "http://tiddlywiki.com/empty.html")
		if err != nil {
			http.Error(w, "Couldn't download an empty wiki!", http.StatusInternalServerError)
			log.Println("Error while downloading empty wiki:", err)
			return
		}

		fmt.Fprintf(w, "Success!")
		return
	}

	wikititle := r.FormValue("wikititle")

	err = renderTemplate(wikitemplate, wikiname, wikititle)
	if err != nil {
		http.Error(w, "Couldn't render the template!", http.StatusInternalServerError)
		log.Println("Error while rendering the template:", err)
		return
	}

	fmt.Fprintf(w, "Success!")
}

func renderTemplate(wikitemplate string, wikiname string, wikititle string) error {
	// Open template file
	tplpath := filepath.Join(cfg.TemplateDir, wikitemplate)
	tplf, err := os.Open(tplpath)
	if err != nil {
		return err
	}
	defer tplf.Close()

	r := bufio.NewReader(tplf)

	// Open wiki file
	wikipath := filepath.Join(cfg.WikiDir, wikiname)
	wikif, err := os.Create(wikipath)
	if err != nil {
		return err
	}
	defer wikif.Close()

	w := bufio.NewWriter(wikif)

	for {
		// read a line
		line, err := r.ReadString('\n')
		if err != nil && err != io.EOF {
			return err
		}
		if len(line) == 0 {
			break
		}

		line = strings.Replace(line, "<!--## Title ##-->", wikititle, -1)
		line = strings.Replace(line, "<!--## Wikiname ##-->", wikiname, -1)
		line = strings.Replace(line, "<!--## Username ##-->", cfg.Username, -1)
		line = strings.Replace(line, "<!--## StoreURL ##-->", serverURL+"/store", -1)

		// write a line
		if _, err := w.WriteString(line); err != nil {
			return err
		}
	}

	if err = w.Flush(); err != nil {
		return err
	}

	return nil
}
