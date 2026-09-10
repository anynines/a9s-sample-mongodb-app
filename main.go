package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"runtime"
	"strings"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"gopkg.in/mgo.v2/bson"
)

type MongoDBCredentials struct {
	Database string `json:"default_database"`
	Uri      string `json:"uri"`
	Cacrt    string `json:"cacrt"`
}

// struct for reading env
type VCAPServices map[string][]struct {
	Credentials MongoDBCredentials `json:"credentials"`
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

	var servicesMap VCAPServices
	err := json.Unmarshal([]byte(os.Getenv("VCAP_SERVICES")), &servicesMap)
	if err != nil {
		log.Println(err)
		return MongoDBCredentials{}, err
	}

	for serviceName, serviceVarList := range servicesMap {
		if !strings.Contains(serviceName, "a9s-mongodb") {
			continue
		}
		if len(serviceVarList) == 0 {
			err = fmt.Errorf("empty list of variables for service %v in env variables", serviceName)
			log.Println(err)
			return MongoDBCredentials{}, err
		}
		log.Printf("Using creds from env for service: %v ", serviceName)
		return serviceVarList[0].Credentials, nil
	}

	err = fmt.Errorf("no matching list environment variables found for mongodb service")
	log.Println(err)
	return MongoDBCredentials{}, err
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

	// a9s nodes use certificates signed by a private CA delivered in the binding.
	if credentials.Cacrt != "" {
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM([]byte(credentials.Cacrt)) {
			err := fmt.Errorf("failed to parse cacrt from service credentials")
			log.Println(err)
			return nil, err
		}
		clientOptions.SetTLSConfig(&tls.Config{RootCAs: pool})
	}

	client, err := mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	err = client.Ping(context.TODO(), nil)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	collection := client.Database(credentials.Database).Collection("posts")

	return collection, err
}

func clearDatabase(w http.ResponseWriter, r *http.Request) {
	collection, err := GetCollection()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err = collection.Drop(context.TODO()); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
		log.Println(err)
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("Collecting blog posts.\n")

	cursor, err := collection.Find(context.TODO(), bson.M{})
	if err != nil {
		fmt.Println("Finding all documents ERROR:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
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
