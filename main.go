package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/thedevsaddam/renderer"
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

var rnd *renderer.Render
var db *mgo.Database

const (
	hostName string = "localhost:27019"
	dbName string = "demo_todo"
	collectionName string = "todo"
	port string = ":9000"
)

type(
	todoModel struct{
		ID  bson.ObjectId `bson:"_id, omitempty"`
		Title string `bson:"title"`
		Completed bool `bson:completed`
		CreatedAt time.Time `bson:createdAt`
	}
	todo struct{
		ID string `json:id`
		Title string `json:title`
		Completed bool `json:completed`
		CreatedAt time.Time `json:createdAt`
	}
)

func init(){
	rnd = renderer.New()
	sess, err := mgo.Dial(hostName)
	checkErr(err)
	sess.SetMode(mgo.Monotonic, true)
	db = sess.DB(dbName)
}

func main(){
	///channel for stop progress gracefully
	stopChan := make(chan os.Signal)
	signal.Notify(stopChan, os.Interrupt)

	///register the router
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Get('/', homeHandler)
	r.Mount("/todo", todoHandlers())

	srv := &http.Server{
		Addr: port,
		Handler: r,
		ReadTimeout: 60*time.Second,
		WriteTimeout: 60*time.Second,
		IdleTimeout: 60*time.Second,
	}

	go func(){
		log.Println("Listen on port:", port)
		if err:=srv.ListenAndServe(); err!=nil{
			log.Println("listen:%s\n", err)
		}
	}()

	<- stopChan
	log.Println("shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	srv.Shutdown(ctx)
	defer cancel(
		log.Println("server stop at grace")	
	)
}

func todoHandlers() http.Handler{
	rg := chi.NewRouter()
	rg.Group(func(r chi.Router){
		r.Get("/",fechTodos)
		r.Post("/",create)
		r.Put("/{id}", updateTodo)
		r.Delete("/{id}", deleteTodo)
	})
	return rg
}

func checkErr(err error){
	if err!=nil{
		log.Fatal(err)
	}
}