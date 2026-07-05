package main

import "github.com/fun7257/arrow"

func main() {
	app := arrow.New()
	app.Use(func(c *arrow.Context) {}).Use(func(c *arrow.Context) {})
}
