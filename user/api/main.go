package main

import (
    "fmt"
    "net/http"
	"user/constant"
)

func handler(w http.ResponseWriter, r *http.Request) {
    fmt.Fprint(w, "Succcess")
}

func main() {
    http.HandleFunc("/ping", handler)
    fmt.Printf("Server is running on %v",constant.Port1)
    http.ListenAndServe(constant.Port1, nil)
}
