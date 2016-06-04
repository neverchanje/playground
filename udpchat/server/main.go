package main

func main() {

	hub, err := NewHub()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	defer hub.Close()

	hub.RunLoop()
}
