/*

coolhttpd: static file server
author: boot-error

*/

package main

import (
	"encoding/base64"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Host      string
	Port      uint
	Username  string
	Passwd    string
	Directory string
}

type File struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	IsDir    bool   `json:"isdir"`
	Modified string `json:"modified"`
	Size     int64  `json:"size"`
	Contents []File `json:"contents"`
}

func getDirList(rootdir string) (f File, err error) {

	var fdata os.FileInfo
	fdata, err = os.Stat(rootdir)

	if err != nil {
		fmt.Println(err)
		return
	}

	f.Name = fdata.Name()
	f.Path = rootdir
	f.Size = fdata.Size()
	f.Modified = fdata.ModTime().Format(time.UnixDate)

	if fdata.IsDir() {

		f.IsDir = true
		files, _ := ioutil.ReadDir(rootdir)

		for _, file := range files {
			fc, _ := getDirList(path.Join(rootdir, file.Name()))
			f.Contents = append(f.Contents, fc)
		}
	}

	return

}

func renderDirectory(f File, w http.ResponseWriter) {

	fmt.Fprintf(w, "<details style=\"margin-left:10px;\">")
	fmt.Fprintf(w, "<summary>%s</summary>", f.Name)
	for _, cf := range f.Contents {
		if cf.IsDir {
			renderDirectory(cf, w)
		} else {
			resourceURL := path.Join("/file/", strings.TrimPrefix(cf.Path, config.Directory))
			fmt.Fprintf(w, "<p style=\"margin: 0px;\"><a href=\"%s\" style=\" margin-left:10px\">%s</a></p>", resourceURL, cf.Name)
		}
	}
	fmt.Fprintf(w, "</details>")

}

func directoryListingHandler(w http.ResponseWriter, r *http.Request) {

	f, err := getDirList(config.Directory)
	w.Header().Add("Content-Type", "text/html; charset=utf-8")

	if err != nil {
		http.Error(w, "Error while indexing directory", http.StatusInternalServerError)
	}

	if f.IsDir {

		fmt.Fprintf(w, "<h3>Directory %s</h3>", f.Name)
		renderDirectory(f, w)

	} else {

		http.Redirect(w, r, f.Name, http.StatusTemporaryRedirect)
	}

}

func fileServingHandler(w http.ResponseWriter, r *http.Request) {

	// fmt.Printf("Serving file %s\n", path.Join(config.Directory, strings.TrimPrefix(r.URL.Path, "/file/")))
	http.ServeFile(w, r, path.Join(config.Directory, strings.TrimPrefix(r.URL.Path, "/file/")))
}

func AuthMiddleware(handlerFunc http.HandlerFunc) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {

		authParams, chk := r.Header["Authorization"]
		if !chk {
			w.Header().Add("WWW-authenticate", "Basic")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		authData := strings.Split(authParams[0], " ")
		if authData[0] == "Basic" {
			decodedAuthData, err := base64.StdEncoding.DecodeString(authData[1])
			if err != nil {
				fmt.Println(err)
				http.Error(w, "Unauthorized, Failed to decode auth credentials", http.StatusUnauthorized)
				return
			}
			authValues := strings.Split(string(decodedAuthData), ":")
			if authValues[0] == config.Username && authValues[1] == config.Passwd {
				handlerFunc(w, r)
			} else {
				fmt.Println(authValues)
				http.Error(w, "Unauthorized, wrong credentials", http.StatusUnauthorized)
				return
			}
		} else {
			fmt.Println(authData[0])
			http.Error(w, "Unauthorized, Couldn't understand auth header", http.StatusUnauthorized)
			return
		}
	}
}

func LoggingMiddleware(handlerFunc http.HandlerFunc) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		handlerFunc(w, r)
		fmt.Println(
			r.RemoteAddr,
			r.Method,
			r.Proto,
			r.Header["User-Agent"],
			r.URL,
		)
	}
}

var config = Config{
	Username:  "test",
	Passwd:    "123",
	Directory: ".",
	Host:      "0.0.0.0",
	Port:      8000,
}

func setupConfig() {

	flag.StringVar(&config.Username, "username", config.Username, "Username for the Basic Authentication")
	flag.StringVar(&config.Passwd, "passwd", config.Passwd, "Password for the Basc Authentication")
	flag.StringVar(&config.Host, "host", config.Host, "ip address for the server")
	flag.UintVar(&config.Port, "port", config.Port, "server port")
	flag.StringVar(&config.Directory, "dir", config.Directory, "directory to host")

	flag.Parse()
}

func main() {

	setupConfig()

	http.Handle("/", LoggingMiddleware(AuthMiddleware(directoryListingHandler)))
	http.Handle("/file/", LoggingMiddleware(AuthMiddleware(fileServingHandler)))

	http.ListenAndServe(strings.Join([]string{config.Host, strconv.Itoa(int(config.Port))}, ":"), nil)
}
