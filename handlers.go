package main

import (
    "fmt"
    "net/http"
)


func handlerHealthz(w http.ResponseWriter, req *http.Request) {
    w.Header().Set("Content-Type", "text/plain; charset=utf-8")
    w.WriteHeader(200)
    w.Write([]byte("200 OK"))
}

func (cfg *apiConfig) handlerHits(w http.ResponseWriter, req *http.Request) {
    fmt.Printf("Hits: %d\n", cfg.fileserverHits.Load())
    w.Write([]byte(fmt.Sprintf("Hits: %d", cfg.fileserverHits.Load())))
}

func (cfg *apiConfig) handlerReset(w http.ResponseWriter, req *http.Request) {
    _ = cfg.fileserverHits.Swap(int32(0))
    fmt.Printf("Site hits reset!\n")
}


func (cfg *apiConfig) handlerAppFunc(w http.ResponseWriter, req *http.Request) {
    _ = cfg.fileserverHits.Add(int32(1))
    fmt.Printf("Works\n")
}
