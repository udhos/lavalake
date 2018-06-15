package main

import (
	"log"
	"os"
)

func main() {
	me := os.Args[0]

	if len(os.Args) < 3 {
		log.Printf("%s: insufficient arguments", me)
		log.Printf("usage:   %s list|pull|push cloud [args]", me)
		log.Printf("example: %s list           aws", me)
		log.Printf("example: %s pull           aws   allow-http vpc-000", me)
		os.Exit(1)
	}

	cmd := os.Args[1]
	cloud := os.Args[2]
	args := os.Args[3:]

	switch {
	case cloud == "aws":
		if err := cloudAws(me, cmd, args); err != nil {
			log.Printf("%s: %v", me, err)
			os.Exit(3)
		}
	case cloud == "azure":
		if err := cloudAzure(me, cmd, args); err != nil {
			log.Printf("%s: %v", me, err)
			os.Exit(3)
		}
	case cloud == "openstack":
		if err := cloudOpenstack(me, cmd, args); err != nil {
			log.Printf("%s: %v", me, err)
			os.Exit(3)
		}
	default:
		log.Printf("%s: cloud not supported: %s", me, cloud)
		os.Exit(2)
	}
}
