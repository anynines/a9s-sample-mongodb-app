package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"runtime"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"gopkg.in/mgo.v2/bson"
)

type MongoDBCredentials struct {
	Database string `json:"default_database"`
	Uri      string `json:"uri"`
}

// struct for reading env
type VCAPServices struct {
	MongoDB []struct {
		Credentials MongoDBCredentials `json:"credentials"`
	} `json:"mongodb40"`
}

type BlogPost struct {
	Title       string `bson: title`
	Description string `bson: description`
}

// template store
var templates map[string]*template.Template

// fill template store
func initTemplates() {
	if templates == nil {
		templates = make(map[string]*template.Template)
	}
	templates["index"] = template.Must(template.ParseFiles("templates/index.html", "templates/base.html"))
	templates["new"] = template.Must(template.ParseFiles("templates/new.html", "templates/base.html"))
}

func getCredentials() (MongoDBCredentials, error) {
	// Kubernetes
	if os.Getenv("VCAP_SERVICES") == "" {
		uri := os.Getenv("MONGODB_URI")
		if len(uri) < 1 {
			err := fmt.Errorf("Environment variable MONGODB_URI missing.")
			log.Println(err)
			return MongoDBCredentials{}, err
		}
		database := os.Getenv("MONGODB_DATABASE")
		if len(database) < 1 {
			err := fmt.Errorf("Environment variable MONGODB_DATABASE missing.")
			log.Println(err)
			return MongoDBCredentials{}, err
		}

		credentials := MongoDBCredentials{
			Uri:      uri,
			Database: database,
		}
		return credentials, nil
	}

	var s VCAPServices
	err := json.Unmarshal([]byte(os.Getenv("VCAP_SERVICES")), &s)
	if err != nil {
		log.Println(err)
		return MongoDBCredentials{}, err
	}

	return s.MongoDB[0].Credentials, nil
}

func renderTemplate(w http.ResponseWriter, name string, template string, viewModel interface{}) {
	tmpl, _ := templates[name]
	err := tmpl.ExecuteTemplate(w, template, viewModel)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func GetCollection() (*mongo.Collection, error) {
	credentials, err := getCredentials()
	if err != nil {
		return nil, err
	}

	clientOptions := options.Client().ApplyURI(credentials.Uri)
	client, err := mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	err = client.Ping(context.TODO(), nil)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	collection := client.Database(credentials.Database).Collection("posts")

	return collection, err
}

func clearDatabase(w http.ResponseWriter, r *http.Request) {
	collection, err := GetCollection()
	if err != nil {
		log.Fatal(err)
		return
	}

	if err = collection.Drop(context.TODO()); err != nil {
		log.Fatal(err)
	}
}

// Create new Blog post
func createBlogPost(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	post := BlogPost{
		Title:       r.PostFormValue("title"),
		Description: r.PostFormValue("description"),
	}

	http.Redirect(w, r, "/", 302)

	collection, err := GetCollection()
	if err != nil {
		log.Fatal(err)
		return
	}

	res, err := collection.InsertOne(context.TODO(), post)
	if err != nil {
		log.Printf("Failed to create new blog post with title %v and description %v ; err = %v", post.Title, post.Description, err)
		return
	}
	log.Println("Inserted document: ", res.InsertedID)
}

func newBlogPost(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, "new", "base", nil)
}

func renderBlogPosts(w http.ResponseWriter, r *http.Request) {
	blogposts := make([]BlogPost, 0)

	collection, err := GetCollection()
	if err != nil {
		log.Fatal(err)
		return
	}

	log.Printf("Collecting blog posts.\n")

	cursor, err := collection.Find(context.TODO(), bson.M{})
	if err != nil {
		fmt.Println("Finding all documents ERROR:", err)
	}

	for cursor.Next(context.TODO()) {
		var post BlogPost
		err := cursor.Decode(&post)

		if err != nil {
			fmt.Println("cursor.Next() error:", err)
		} else {
			blogposts = append(blogposts, post)
		}
	}

	renderTemplate(w, "index", "base", blogposts)
}

func main() {
	log.Println(runtime.Version())

	initTemplates()

	port := "3000"
	if port = os.Getenv("PORT"); len(port) == 0 {
		port = "3000"
	}

	http.Handle("/public/", http.StripPrefix("/public/", http.FileServer(http.Dir("./public"))))

	http.HandleFunc("/", renderBlogPosts)
	http.HandleFunc("/blog-posts/new", newBlogPost)
	http.HandleFunc("/blog-posts/create", createBlogPost)
	http.HandleFunc("/clear", clearDatabase)

	log.Printf("Listening on :%v\n", port)
	http.ListenAndServe(fmt.Sprintf(":%s", port), nil)
}
