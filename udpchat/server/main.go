package main

import (
	"fmt"
	"os"

	"github.com/neverchanje/unplayground/udpchat"
)

func main() {

	hub, err := udpchat.NewHub()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	hub.RunLoop()
}
