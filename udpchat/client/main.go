package main

import (
	"bufio"
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/neverchanje/unplayground/udpchat"
)

func main() {

	fmt.Println("udpchat (" + time.Now().Format(time.UnixDate) + ")")
	fmt.Println("[" + runtime.GOOS + " " + runtime.GOARCH + "]")
	fmt.Println("Type \"help\" for more information.")

	fmt.Print("\nPlease enter your username: ")
	username, _, _ := bufio.NewReader(os.Stdin).ReadLine()

	client, err := udpchat.NewClient(string(username))
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer client.Close()

	fmt.Println("Successful launch!")
	client.RunLoop()
}
