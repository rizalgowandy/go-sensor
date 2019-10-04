package main

import (
    "fmt"
    "time"
    "log"
    "net/http"
    "encoding/json"
    "github.com/gorilla/mux"
    instana "github.com/instana/go-sensor"
)


var sensor = instana.NewSensor("Gorilla")


// HTTP Handler
func healthHandler(w  http.ResponseWriter, req *http.Request) {
    json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

//API Handler
func apiHandler(w http.ResponseWriter, req *http.Request) {
    vars := mux.Vars(req)
    bar := vars["bar"]
    time.Sleep(500 * time.Millisecond)
    log.Println("foo =", bar)
    json.NewEncoder(w).Encode(map[string]string{"bar": bar})
}

//SKU Handler
func skuHandler(w http.ResponseWriter, req *http.Request) {
    vars := mux.Vars(req)
    id := vars["id"]
    time.Sleep(300 * time.Millisecond)
    log.Println("id =", id)
    json.NewEncoder(w).Encode(map[string]string{"id": id})
}

// Explosion
func bangHandler(w http.ResponseWriter, req *http.Request) {
    time.Sleep(200 * time.Millisecond)
    w.WriteHeader(http.StatusInternalServerError)
    fmt.Fprintf(w, "BANG")
}

// Not found
func makeNotFoundHandler() http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
        w.WriteHeader(http.StatusNotFound)
        fmt.Fprintf(w, "This is not the page you are looking for")
    })
}

func main() {
    fmt.Println("Starting...")

    router := mux.NewRouter()

    router.HandleFunc("/health", healthHandler).Methods("GET")
    router.HandleFunc("/api/{bar}", apiHandler).Methods("GET")
    router.HandleFunc("/sku/{id:[0-9]+}", skuHandler).Methods("GET")
    router.HandleFunc("/bang", bangHandler).Methods("GET")

    router.NotFoundHandler = makeNotFoundHandler()

    // wrap the mux
    sensor.GorillaWrap(router)

    srv := &http.Server{
        Handler: router,
        Addr: ":8080",
        WriteTimeout: 15 * time.Second,
        ReadTimeout: 15 * time.Second,
    }

    log.Fatal(srv.ListenAndServe())
}

