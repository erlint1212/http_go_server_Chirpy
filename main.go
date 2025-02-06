package main

import (
    "fmt"
    "net/http"
    "log"
)

func check(err error) {
    if err != nil {
        log.Fatalf("error: %v\n", err)
    }
}

func main() {
    mux := http.NewServeMux()
    srv := http.Server{
        Addr: ":8080",
        Handler: mux,
    }

    err := srv.ListenAndServe()
    check(err)

    fmt.Printf("works %v\n", srv)
}
