package main

import(
	"fmt"
	"net/http"
	"io/ioutil"
	"math/rand"
	"time"
	"strings"
//	"regexp"
)

func shuffle(w http.ResponseWriter, r *http.Request) {
	p := 0
	if rand.Float32() > 0.5 {
		p1 := rand.Perm(9)
		for _, p = range p1 {
			fmt.Fprint(w, fmt.Sprint(p)+"|")
		}
		p2 := rand.Perm(8)
		for _, p = range p2 {
			fmt.Fprint(w, fmt.Sprint(p)+"|")
		}
		fmt.Println(fmt.Sprint(p1)+"; "+fmt.Sprint(p2))
	} else {
		p1 := rand.Perm(8)
		for _, p = range p1 {
			fmt.Fprint(w, fmt.Sprint(p)+"|")
		}
		p2 := rand.Perm(9)
		for _, p = range p2 {
			fmt.Fprint(w, fmt.Sprint(p)+"|")
		}
		fmt.Println(fmt.Sprint(p1)+"; "+fmt.Sprint(p2))
	}
}

var pPathLength = len("/p/")

func pngs(w http.ResponseWriter, r *http.Request) {
	pic := r.URL.Path[pPathLength:]
	http.ServeFile(w, r, pic)
}

func client(w http.ResponseWriter, r *http.Request) {
	page, _ := ioutil.ReadFile("index.html")
	fmt.Fprint(w, string(page))
}

func devclient(w http.ResponseWriter, r *http.Request) {
	page, _ := ioutil.ReadFile("blickles-dev.html")
	fmt.Fprint(w, string(page))
}

func clientmaker() {
	wholefile, _ := ioutil.ReadFile("blickles-dev.html")
	sFile := string(wholefile)
	sFileByLine := strings.Split(sFile, "\n")
	var parsedBFile []byte
	for _, l := range sFileByLine {
		if strings.Index(l,"DBGGR") == -1 {
			parsedBFile = append(parsedBFile, []byte(fmt.Sprintln(l))...)
		}
	}
	ioutil.WriteFile("index.html", parsedBFile, 0600)
}

func main() {
	t := time.Now()
	rand.Seed(int64(t.Nanosecond()))
	clientmaker()
	http.HandleFunc("/pieces", shuffle)
	http.HandleFunc("/", client)
	http.HandleFunc("/index.html", client)
	http.HandleFunc("/p/", pngs)
	http.HandleFunc("/dev", devclient)
	http.ListenAndServe(":8080",nil)
}
