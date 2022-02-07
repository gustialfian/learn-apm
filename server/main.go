package main

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"go.elastic.co/apm"
	"go.elastic.co/apm/module/apmgorilla"
	"go.elastic.co/apm/module/apmhttp"
	"go.elastic.co/apm/module/apmlogrus"
)

func main() {
	fmt.Println("server start")

	r := mux.NewRouter()
	apmgorilla.Instrument(r)

	r.HandleFunc("/hello", helloHandler)
	r.HandleFunc("/error", errorHandler)
	r.HandleFunc("/log", logHandler())
	r.HandleFunc("/send", sendHandler)

	fmt.Println("OK")

	file, err := os.OpenFile("out.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err == nil {
		logrus.SetOutput(file)
	}
	defer file.Close()

	log.Fatal(http.ListenAndServe(":8080", r))
}

func helloHandler(w http.ResponseWriter, r *http.Request) {
	span, _ := apm.StartSpan(r.Context(), "hello", "handler")
	defer span.End()

	fmt.Fprintf(w, "hello\n")
}

func errorHandler(w http.ResponseWriter, r *http.Request) {
	span, ctx := apm.StartSpan(r.Context(), "error", "handler")
	defer span.End()

	apm.CaptureError(ctx, errors.New("ooops")).Send()

	fmt.Fprintf(w, "error\n")
}

func sendHandler(w http.ResponseWriter, r *http.Request) {
	span, ctx := apm.StartSpan(r.Context(), "send", "handler")
	defer span.End()
	client := apmhttp.WrapClient(&http.Client{})

	work := sendWork(ctx, client)
	help := sendHelp(ctx, client)
	if err := sendNone(ctx, client); err != nil {
		fmt.Printf("err: %s", err.Error())
		apm.CaptureError(ctx, fmt.Errorf("send none fail: %w", err)).Send()
	}

	result := fmt.Sprintf("work: %s, help: %s", string(work), string(help))

	fmt.Fprintf(w, "work done: %v", result)
}

func logHandler() func(w http.ResponseWriter, r *http.Request) {
	logrus.AddHook(&apmlogrus.Hook{})
	logrus.SetFormatter(&logrus.JSONFormatter{
		FieldMap: logrus.FieldMap{
			logrus.FieldKeyTime: "@timestamp",
			logrus.FieldKeyMsg:  "message",
		},
	})
	logrus.SetLevel(logrus.TraceLevel)

	return func(w http.ResponseWriter, r *http.Request) {
		span, _ := apm.StartSpan(r.Context(), "send", "handler")
		defer span.End()

		traceContextFields := apmlogrus.TraceContext(r.Context())
		logrus.WithFields(traceContextFields).Debug("handling request")
		logrus.WithFields(traceContextFields).Info("this is info yoo")

		fmt.Fprint(w, "logging done")

	}
}

func sendWork(ctx context.Context, client *http.Client) string {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost:8081/work", nil)
	if err != nil {
		panic(err)
	}
	res, err := client.Do(req)
	if err != nil {
		panic(err)
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		panic(err)
	}
	defer res.Body.Close()

	return string(body)
}

func sendHelp(ctx context.Context, client *http.Client) string {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost:8081/help", nil)
	if err != nil {
		panic(err)
	}
	res, err := client.Do(req)
	if err != nil {
		panic(err)
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		panic(err)
	}
	defer res.Body.Close()

	return string(body)
}

func sendNone(ctx context.Context, client *http.Client) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost:8082/none", nil)
	if err != nil {
		return err
	}
	res, err := client.Do(req)
	if err != nil {
		return err
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	fmt.Println(string(body))

	return nil
}
