package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"

	"otelPlayground/tracing"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux"
	"go.opentelemetry.io/contrib/instrumentation/go.mongodb.org/mongo-driver/mongo/otelmongo"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

const (
	service     = "otelPlayground"
	environment = "localTesting"
	id          = 1
)

var mongoClient *mongo.Client
var httpClient *http.Client

func main() { //Export traces to Jaeger
	tp, err := tracing.InitProvider(service)
	if err != nil {
		log.Fatal(err)
	}
	defer tracing.Shutdown(tp)

	httpClient = &http.Client{
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	}

	connectMongo()
	startWebServerMux()
}

func connectMongo() {
	opts := options.Client()
	//Mongo OpenTelemetry instrumentation
	opts.Monitor = otelmongo.NewMonitor()
	opts.ApplyURI("mongodb://0.0.0.0:27017")
	if opts.Validate() != nil {
		log.Fatal("MongoDB client connection failed")
	}
	mongoClient, _ = mongo.Connect(context.Background(), opts)
	//Seed the database with some todo's
	docs := []interface{}{
		bson.D{{"id", "1"}, {"title", "Buy groceries"}},
		bson.D{{"id", "2"}, {"title", "install Aspecto.io"}},
		bson.D{{"id", "3"}, {"title", "Buy dogz.io domain"}},
	}
	mongoClient.Database("todo").Collection("todos").InsertMany(context.Background(), docs)
}

func startWebServerMux() {
	router := mux.NewRouter()
	router.Use(otelmux.Middleware("otelMuxExample"))

	// forwards request to /gettodo
	router.HandleFunc("/todo", func(w http.ResponseWriter, r *http.Request) {
		req, err := http.NewRequestWithContext(
			r.Context(),
			"GET",
			"http://localhost:8080/gettodo",
			nil,
		)
		req = req.WithContext(r.Context())

		if err != nil {
			fmt.Println(err.Error())
		}

		res, err := httpClient.Do(req)

		if err != nil {
			fmt.Println(err.Error())
		}
		defer res.Body.Close()
		body, _ := io.ReadAll(res.Body)

		fmt.Fprintf(w, "%s", string(body))
	})
	// fetches data from data bank
	router.HandleFunc("/gettodo", func(w http.ResponseWriter, r *http.Request) {
		collection := mongoClient.Database("todo").Collection("todos")
		cur, findErr := collection.Find(r.Context(), bson.D{})
		if findErr != nil {
			fmt.Fprintf(w, "Error")
			return
		}
		results := make([]interface{}, 0)
		curErr := cur.All(r.Context(), &results)
		if curErr != nil {
			fmt.Fprintf(w, "Error")
			return
		}
		fmt.Fprintf(w, "Hello")
	})

	log.Print("Running server on 0.0.0.0:8080")
	http.Handle("/", router)
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal(err)
	}
}
