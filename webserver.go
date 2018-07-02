// Webserver with built-in health checks for static content Example
// John Radosta - Cloud Solutions Architect | Dito (http://www.ditoweb.com)

package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

var (
	url      = "https://s3.amazonaws.com/react-web/mona-lisa.jpg"
	styles   = "https://s3.amazonaws.com/react-web/styles.css"
	url2     = ""
	styles2  = ""
	failover = false
	tmpl     *template.Template
)

// Original Image:
// https://images.pexels.com/photos/104827/cat-pet-animal-domestic-104827.jpeg?auto=compress&cs=tinysrgb&h=350

func init() {
	if len(os.Args) > 1 {
		url2 = os.Args[1]
		styles2 = os.Args[2]
	}
	tmpl = template.Must(template.ParseGlob("html/*"))
}

func SetMyCookie(response http.ResponseWriter) {
	// Add a simplistic cookie to the response.
	cookie := http.Cookie{Name: "testcookiename", Value: "testcookievalue"}
	http.SetCookie(response, &cookie)
}

// Respond to URLs of the form /generic/...
func GenericHandler(response http.ResponseWriter, request *http.Request) {

	// Set cookie and MIME type in the HTTP headers.
	SetMyCookie(response)
	response.Header().Set("Content-type", "text/plain")

	// Parse URL and POST data into the request.Form
	err := request.ParseForm()
	if err != nil {
		http.Error(response, fmt.Sprintf("error parsing url %v", err), 500)
	}

	// Send the text diagnostics to the client.
	fmt.Fprint(response, "WebServerStatus says ... \n")
	fmt.Fprintf(response, " request.Method     '%v'\n", request.Method)
	fmt.Fprintf(response, " request.RequestURI '%v'\n", request.RequestURI)
	fmt.Fprintf(response, " request.URL.Path   '%v'\n", request.URL.Path)
	fmt.Fprintf(response, " request.Form       '%v'\n", request.Form)
	fmt.Fprintf(response, " request.Cookies()  '%v'\n", request.Cookies())
}

// Respond to the URL /home with an html home page
func HomeHandler(response http.ResponseWriter, request *http.Request) {
	response.Header().Set("Content-type", "text/html")
	if failover == false {
		source := map[string]interface{}{
			"img": url,
			"css": styles,
		}
		err := tmpl.ExecuteTemplate(response, "home.tmpl", source)
		if err != nil {
			panic(err)
		}
	} else {
		source := map[string]interface{}{
			"img": url2,
			"css": styles2,
		}
		err := tmpl.ExecuteTemplate(response, "home.tmpl", source)
		if err != nil {
			panic(err)
		}
	}
}

func main() {
	sites := []string{
		url,
		styles,
	}

	status := make(chan string)
	ticker := time.NewTicker(10 * time.Second)
	failoverChan := make(chan bool)
	var buffer int

	go func(s []string) {
		for {
			select {
			case <-ticker.C:
				buffer = len(s)
				checkLinks(s, status, failoverChan)

			case <-failoverChan:
				failover = true
			}
		}
	}(sites)

	port := 8097
	portstring := strconv.Itoa(port)

	mux := http.NewServeMux()
	mux.Handle("/", http.HandlerFunc(HomeHandler))
	mux.Handle("/generic", http.HandlerFunc(GenericHandler))

	go func() {
		for {
			fmt.Println(<-status)
		}
	}()

	go func() {
		log.Print("Listening on port " + portstring + " ... ")
		err := http.ListenAndServe(":"+portstring, mux)
		if err != nil {
			log.Fatal("ListenAndServe error: ", err)
		}
	}()

	// Blocks on the main thread
	select {}
}

func checkLinks(sites []string, status chan string, failoverChan chan bool) {
	for _, site := range sites {
		go func(s string) {
			resp, err := http.Get(s)
			if err != nil {
				fmt.Println("Error Fetching Object! Failing over... ", err)
				failoverChan <- true
			}
			if resp.StatusCode != 200 {
				fmt.Println("Error Fetching Object! Failing over... ", " - DOWN")
				failoverChan <- true
			} else {
				status <- s + " - OK"
				failover = false
			}
		}(site)
	}
}
