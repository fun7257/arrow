package main

import "github.com/fun7257/arrow"

func main() {
	app := arrow.New()
	app.Group("/api").Use(func(c *arrow.Context) {})
}
