// main implements the lake utility.
package main

import (
	"fmt"
	"log"
	"os"
)

var debug bool

func main() {
	me := os.Args[0]

	if len(os.Args) < 3 {
		fmt.Printf("%s: insufficient arguments\n", me)
		fmt.Println()
		fmt.Printf("usage:   %s list|pull|push cloud [args]\n", me)
		fmt.Println()
		fmt.Printf("example: %s list aws\n", me)
		fmt.Printf("example: %s pull aws group1 vpc-id > group1.yaml\n", me)
		fmt.Printf("example: %s push aws group2 vpc-id < group2.yaml\n", me)
		fmt.Println()
		fmt.Printf("example: %s list azure\n", me)
		fmt.Printf("example: %s pull azure group1 resource-group-name > group1.yaml\n", me)
		fmt.Printf("example: %s push azure group2 resource-group-name location < group1.yaml\n", me)
		fmt.Println()
		fmt.Printf("example: %s list openstack\n", me)
		fmt.Printf("example: %s pull openstack group1 > group1.yaml\n", me)
		fmt.Printf("example: %s push openstack group2 < group1.yaml\n", me)

		os.Exit(1)
	}

	debug = os.Getenv("DEBUG") != ""
	log.Printf("DEBUG=[%s] debug=%v", os.Getenv("DEBUG"), debug)

	cmd := os.Args[1]
	cloud := os.Args[2]
	args := os.Args[3:]

	switch {
	case cloud == "aws":
		if err := cloudAws(me, cmd, cloud, args); err != nil {
			log.Printf("%s: %v", me, err)
			os.Exit(3)
		}
	case cloud == "azure":
		if err := cloudAzure(me, cmd, cloud, args); err != nil {
			log.Printf("%s: %v", me, err)
			os.Exit(3)
		}
	case cloud == "openstack":
		if err := cloudOpenstack(me, cmd, cloud, args); err != nil {
			log.Printf("%s: %v", me, err)
			os.Exit(3)
		}
	default:
		log.Printf("%s: cloud not supported: %s", me, cloud)
		os.Exit(2)
	}
}
