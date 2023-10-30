package main

import (
    "fmt"
    "net/http"
	"user/constant"
)

// func handler(w http.ResponseWriter, r *http.Request) {
//     fmt.Fprint(w, "Hello, World!")
//     for i:=1;i<=3;i++{
//         fmt.Fprint(w,"hello")
//     }
// }
func handler(w http.ResponseWriter, r *http.Request) {
    fmt.Fprint(w, "success")

    for i := 1; i <= 3; i++ {
        fmt.Println(" hello")
    }
}


func main() {
    http.HandleFunc("/ping", handler)
    fmt.Printf("Server is running on %v",constant.Port2)
    http.ListenAndServe(constant.Port2, nil)
}
