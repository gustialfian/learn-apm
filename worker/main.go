package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"go.elastic.co/apm"
	"go.elastic.co/apm/module/apmgorilla"
)

func main() {
	fmt.Println("worker start")

	r := mux.NewRouter()
	apmgorilla.Instrument(r)

	r.HandleFunc("/work", workHandler)
	r.HandleFunc("/help", accidentHandler)

	fmt.Println("OK")
	log.Fatal(http.ListenAndServe(":8081", r))
}

func workHandler(w http.ResponseWriter, r *http.Request) {
	span, _ := apm.StartSpan(r.Context(), "work", "handler")
	defer span.End()

	fmt.Fprint(w, "worked")
}

func accidentHandler(w http.ResponseWriter, r *http.Request) {
	span, ctx := apm.StartSpan(r.Context(), "accident", "handler")
	defer span.End()

	apm.CaptureError(ctx, errors.New("worker fail")).Send()

	fmt.Fprint(w, "fail")

}
