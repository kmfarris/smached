package main

import (
	"encoding/json"
	"fmt"
	"github.com/julienschmidt/httprouter"
	"io/ioutil"
	"log"
	"net/http"
	smached "smached/caching"
)

func homePage(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	fmt.Fprint(w, "Smached!")
	m, c := smached.ShowServerStats()
	fmt.Printf("%v MiB | Record Count %v \n\r", m/1024/1024, c)

}

func handleRequest() {

	router := httprouter.New()
	//router.GET("/api/record/:key",BasicAuth(findRecord, smached.GetAuthToken()))
	router.GET("/api/record/:key", BasicAuth(findRecord, smached.GetAuthToken()))
	router.GET("/api/record/", BasicAuth(findRecord, smached.GetAuthToken()))
	router.GET("/api/stats", BasicAuth(homePage, smached.GetAuthToken()))
	router.POST("/api/record", BasicAuth(addRecord, smached.GetAuthToken()))
	fileServer := http.FileServer(http.Dir("./web"))
	http.Handle("/", fileServer)
	http.Handle("/api/", router)
	log.Fatal(http.ListenAndServe(":10000", nil))

}

func BasicAuth(h httprouter.Handle, authToken string) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		// Get the Basic Authentication credentials
		//user, password, hasAuth := r.BasicAuth()
		passedAuthToken := r.Header.Get("authToken")
		if passedAuthToken == authToken {
			// Delegate request to the given handle
			log.Printf("Authentication success from: %v", r.RemoteAddr)
			h(w, r, ps)
		} else {
			// Request Basic Authentication otherwise
			w.Header().Set("WWW-Authenticate", "Basic realm=Restricted")
			log.Printf("Authentication failure from: %v", r.RemoteAddr)
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		}
	}
}

func findRecord(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	key := params.ByName("key")
	if key == "" {
		jsonResponse, err := json.Marshal(smached.GetAll())
		if err != nil {
			log.Fatal(err)
			return
		}
		writeResponse(w, jsonResponse)
		return
	}
	record := smached.Find(key)
	if record == nil {
		writeResponse(w, []byte("{}"))
		return
	}
	jsonResponse, err := json.Marshal(record.Value)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeResponse(w, jsonResponse)
}

func writeResponse(w http.ResponseWriter, payload []byte) {
	w.Header().Set("Content-Type", "application/json")
	_, err := w.Write(payload)
	if err != nil {
		log.Fatal(err)
		return
	}
}

func addRecord(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	var body smached.Record
	b, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()

	var v map[string]interface{}
	json.Unmarshal([]byte(b), &v)
	body.Value = v["value"]
	body.Expires = v["expires"]
	body.ForceDb = v["forcedb"]
	if v["ttl"] != nil {
		body.Ttl = fmt.Sprintf("%v", v["ttl"])
	} else {
		body.Ttl = ""
	}

	if err != nil {
		log.Fatal(err)
	}

	jsonResponse, err := json.Marshal(smached.Add(body))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeResponse(w, jsonResponse)
}

func main() {
	smached.InitSmached()
	handleRequest()
}
