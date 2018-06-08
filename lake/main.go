package main

import (
	"log"
	"os"
)

func main() {
	me := os.Args[0]

	if len(os.Args) < 4 {
		log.Printf("%s: insufficient arguments", me)
		log.Printf("usage:   %s pull|push cloud name       [args]", me)
		log.Printf("example: %s pull      aws   allow-http vpc-000", me)
		os.Exit(1)
	}

	cmd := os.Args[1]
	cloud := os.Args[2]
	name := os.Args[3]
	args := os.Args[4:]

	switch {
	case cloud == "aws":
		if err := cloudAws(me, cmd, name, args); err != nil {
			log.Printf("%s: %v", me, err)
			os.Exit(3)
		}
	case cloud == "azure":
		if err := cloudAzure(me, cmd, name, args); err != nil {
			log.Printf("%s: %v", me, err)
			os.Exit(3)
		}
	default:
		log.Printf("%s: cloud not supported: %s", me, cloud)
		os.Exit(2)
	}
}
