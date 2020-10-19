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

func homePage(w http.ResponseWriter, r *http.Request, _ httprouter.Params){
	fmt.Fprint(w, "Smached!")
	m, c := smached.ShowServerStats()
	fmt.Printf("%v MiB | Record Count %v \n\r", m,c)

}

func handleRequest(){
	router := httprouter.New()
	router.GET("/api/record/:key",findRecord)
	router.GET("/api/record/",findRecord)
	router.GET("/api/stats", homePage)
	router.POST("/api/record",addRecord)
	fileServer := http.FileServer(http.Dir("./web"))
	http.Handle("/", fileServer)
	http.Handle("/api/", router)
	log.Fatal(http.ListenAndServe(":10000", nil))

}

func findRecord(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	key := params.ByName("key")
	if key == "" {
		jsonResponse, err := json.Marshal(smached.GetAll())
		if err !=nil{
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
	if err!=nil{
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeResponse(w, jsonResponse)
}

func writeResponse(w http.ResponseWriter, payload []byte){
	w.Header().Set("Content-Type", "application/json")
	_, err := w.Write(payload)
	if err != nil{
		log.Fatal(err)
		return
	}
}

func addRecord(w http.ResponseWriter, r *http.Request, _ httprouter.Params){
	var body smached.Record
		b, err := ioutil.ReadAll(r.Body)
		defer r.Body.Close()

		var v map[string]interface{}
		json.Unmarshal([]byte(b),&v)
		body.Value = v["value"]
		body.Expires= v["expires"]
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

func main(){
	smached.InitSmached()
	handleRequest()
}

