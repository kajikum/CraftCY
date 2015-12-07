package main

import (
	"os"
	"log"
	"runtime"

	"github.com/joho/godotenv"
	"github.com/mitchellh/cli"
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	if godotenv.Load() != nil {
		log.Printf("No such .env")
	}

	c := cli.NewCLI("crafty", "1.0.0")
	c.Args = os.Args[1:]

	c.Commands = map[string]cli.CommandFactory{
		"server": func() (cli.Command, error) {
			return &Server{}, nil
		},
	}

	ret, err := c.Run()
	if err != nil {
		log.Fatal(err)
	}
	os.Exit(ret)
}
