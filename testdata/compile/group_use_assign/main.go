package main

import "github.com/fun7257/arrow"

func main() {
	app := arrow.New()
	g := app.Group("/api")
	_ = g.Use(func(c *arrow.Context) {})
}
