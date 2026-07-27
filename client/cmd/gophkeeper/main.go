package main

import (
	_ "gophkeeper/client/cmd/gophkeeper/secret"
	"gophkeeper/client/internal/app"
)

func main() {
	app.InitApp()
	Execute()
}
