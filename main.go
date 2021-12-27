package main

import (
	"context"
	"encoding/json"
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

func homeHandler(w http.ResponseWriter, req *http.Request){
	err := rnd.Template(w, http.StatusOK, []string{"static/home.tpl"}, nil)
	checkErr(err)
}

///why http.ResonseWriter是用值類型、而 *http.Request要用指針類型??
func fetchTodos(w http.ResponseWriter, req *http.Request){
	todos := []todoModel{}

	if err:= db.C(collectionName).Find(bson.M{}).All(&todos); err !=nil{
		rnd.JSON(w, http.StatusProcessing, renderer.M{
			"message":"fail to load todos from database",
			"err":err,
		})
		return
	}

	todoList := []todo{}

	for _, t := range todos{
		todoList = append(todoList,todo{
			ID: t.ID.Hex(),
			Title: t.Title,
			Completed: t.Completed,
			CreatedAt:t.CreatedAt,
		} )
	}

	rnd.JSON(w, http.StatusOK, renderer.M{
		"data":todoList,
	})
}

func updateTodo(w http.ResponseWriter, r *http.Request){
	i := strings.IrimSpace(chi.URLParam(r, "id"))

	if !bson.IsObjectIdHex(id){
		rnd.JSON(w, http.StatusBadRequest, renderer.M{
			"message":"the id is invalid",
		})
		return
	}

	var t todo

	if err:= json.NewDecoder(r.Body).Decode(&t); err!=nil{
		rnd.JSON(w, http.StatusProcessing, err)
		return
	}

	if t.Title==""{
		rnd.JSON(w, http.StatusBadRequest, renderer.M{
			"message":"the title field is required",
		})
		return
	}

	if err:= db.C(collectionName).Update(
		bson.M{"_id":bson.ObjectIdHex((id))},
		bson.M{"title": t.Title, "completed":t.Completed},
	);err==nil {
		rnd.JSON(w, http.StatusProcessing, renderer.M{
			"message":"failed in update todo",
			"error":err,
		})
		return
	}
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