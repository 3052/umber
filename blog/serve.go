package main

import "net/http"

func main() {
   println("localhost:8080")
   http.ListenAndServe(":8080", http.FileServer(http.Dir("")))
}
