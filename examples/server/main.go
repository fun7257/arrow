// A standard Arrow HTTP service example.
//
// Run from the repository root:
//
//	go run ./examples/server
//
// Then try:
//
//	curl http://localhost:8080/health
//	curl http://localhost:8080/api/v1/posts
//	curl -H "Authorization: Bearer demo-token" http://localhost:8080/api/v1/posts/1
//	curl -X POST -H "Authorization: Bearer demo-token" -d '{"title":"hello"}' http://localhost:8080/api/v1/posts
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"

	"github.com/fun7257/arrow"
	"github.com/fun7257/arrow/middleware"
	"github.com/fun7257/arrow/target"
)

func main() {
	app := arrow.New()

	app.Use(middleware.Recover())
	app.Use(middleware.RequestID())
	app.Use(middleware.Logger())

	app.GET("/", home)
	app.GET("/health", health)

	api := app.Group("/api/v1")
	api.Use(requireToken("demo-token"))
	api.GET("/posts", listPosts)
	api.GET("/posts/{id}", showPost)
	api.POST("/posts", createPost)

	addr := listenAddr()
	log.Printf("arrow example listening on %s", addr)
	if err := app.ListenAndServe(addr); err != nil {
		log.Fatal(err)
	}
}

func listenAddr() string {
	if port := os.Getenv("PORT"); port != "" {
		return ":" + port
	}
	return ":8080"
}

func home(c *arrow.Context) {
	target.WritePlain(c, http.StatusOK, "Arrow HTTP example\n")
}

func health(c *arrow.Context) {
	target.OK(c, map[string]string{"status": "ok"})
}

func requireToken(token string) arrow.HandlerFunc {
	return func(c *arrow.Context) {
		auth := c.Request.Header.Get("Authorization")
		if auth != "Bearer "+token {
			_ = target.AbortUnauthorized(c, "unauthorized")
		}
	}
}

type post struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

var (
	postsMu sync.RWMutex
	posts   = []post{
		{ID: "1", Title: "Getting started with Arrow"},
		{ID: "2", Title: "Penetration middleware model"},
	}
	nextPostID = 3
)

func listPosts(c *arrow.Context) {
	postsMu.RLock()
	defer postsMu.RUnlock()
	target.OK(c, posts)
}

func showPost(c *arrow.Context) {
	id := c.Request.PathValue("id")

	postsMu.RLock()
	defer postsMu.RUnlock()
	for _, p := range posts {
		if p.ID == id {
			target.OK(c, p)
			return
		}
	}
	// Handler path: write 404 without aborting (penetration already complete).
	target.NotFound(c, "post not found")
}

func createPost(c *arrow.Context) {
	var in struct {
		Title string `json:"title"`
	}
	if err := json.NewDecoder(c.Request.Body).Decode(&in); err != nil {
		target.BadRequest(c, "invalid json")
		return
	}
	if in.Title == "" {
		target.BadRequest(c, "title is required")
		return
	}

	postsMu.Lock()
	defer postsMu.Unlock()
	p := post{ID: strconv.Itoa(nextPostID), Title: in.Title}
	nextPostID++
	posts = append(posts, p)
	target.Created(c, p)
}