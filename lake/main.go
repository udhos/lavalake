package main

import (
	"log"
	"os"
)

func main() {
	me := os.Args[0]

	if len(os.Args) < 4 {
		log.Printf("%s: insufficient arguments", me)
		log.Printf("usage:   %s pull|push <cloud> name       [args]", me)
		log.Printf("example: %s pull      aws     allow-http", me)
		os.Exit(1)
	}

	cmd := os.Args[1]
	cloud := os.Args[2]
	name := os.Args[3]
	args := os.Args[4:]

	switch {
	case cloud == "aws":
		cloudAws(me, cmd, name, args)
	case cloud == "azure":
		cloudAzure(me, cmd, name, args)
	default:
		log.Printf("%s: cloud not supported: %s", me, cloud)
		os.Exit(2)
	}
}
