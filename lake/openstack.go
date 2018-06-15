package main

import (
	"fmt"
	"log"
	"os"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/security/groups"
	//"github.com/gophercloud/gophercloud/openstack/utils"
)

func cloudOpenstack(me, cmd string, args []string) error {

	switch cmd {
	case "list":
		return listOpenstack(me, cmd)
	case "pull":
		return fmt.Errorf("openstack FIXME WRITEME: cmd=%s", cmd)
	case "push":
		return fmt.Errorf("openstack FIXME WRITEME: cmd=%s", cmd)
	}

	return fmt.Errorf("unsupported aws command: %s", cmd)
}

func showCredentialsOpenstack() {
	cred("OS_REGION_NAME")
}

func listOpenstack(me, cmd string) error {

	showCredentialsOpenstack()

	regionName := os.Getenv("OS_REGION_NAME")
	if regionName == "" {
		return fmt.Errorf("missing env var OS_REGION_NAME")
	}

	opts, err := openstack.AuthOptionsFromEnv()

	provider, err := openstack.AuthenticatedClient(opts)

	client, err := openstack.NewNetworkV2(provider, gophercloud.EndpointOpts{
		Region: regionName,
	})

	allPages, err := groups.List(client, groups.ListOpts{}).AllPages()
	if err != nil {
		panic(err)
	}

	allGroups, err := groups.ExtractGroups(allPages)
	if err != nil {
		panic(err)
	}

	// https://godoc.org/github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/security/groups#SecGroup

	for _, gr := range allGroups {
		log.Printf("name=%s id=%s project=%s description=%s", gr.Name, gr.ID, gr.ProjectID, gr.Description)
	}

	return nil
}
