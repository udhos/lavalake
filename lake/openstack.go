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
		if len(args) < 1 {
			log.Printf("usage: %s %s openstack name", me, cmd)
			return fmt.Errorf("%s %s openstack: missing name", me, cmd)
		}
		name := args[0]
		return pullOpenstack(me, cmd, name)
	case "push":
		return fmt.Errorf("openstack FIXME WRITEME: cmd=%s", cmd)
	}

	return fmt.Errorf("unsupported aws command: %s", cmd)
}

func showCredentialsOpenstack() {
	cred("OS_REGION_NAME")
	cred("OS_TENANT_ID")
	cred("OS_IDENTITY_API_VERSION")
	cred("OS_AUTH_URL")
	cred("OS_TENANT_NAME")
	cred("OS_ENDPOINT_TYPE")
	cred("OS_USERNAME")
	credHide("OS_PASSWORD")
}

func listOpenstack(me, cmd string) error {

	showCredentialsOpenstack()

	regionName := os.Getenv("OS_REGION_NAME")
	if regionName == "" {
		return fmt.Errorf("missing env var OS_REGION_NAME")
	}

	opts, errAuth := openstack.AuthOptionsFromEnv()
	if errAuth != nil {
		return errAuth
	}

	provider, errProv := openstack.AuthenticatedClient(opts)
	if errProv != nil {
		return errProv
	}

	client, errClient := openstack.NewNetworkV2(provider, gophercloud.EndpointOpts{
		Region: regionName,
	})
	if errClient != nil {
		return errClient
	}

	allPages, errList := groups.List(client, groups.ListOpts{}).AllPages()
	if errList != nil {
		return errList
	}

	allGroups, errExtract := groups.ExtractGroups(allPages)
	if errExtract != nil {
		return errExtract
	}

	// https://godoc.org/github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/security/groups#SecGroup

	for _, gr := range allGroups {
		fmt.Printf("name=%s id=%s project=%s description=%s\n", gr.Name, gr.ID, gr.ProjectID, gr.Description)
	}

	return nil
}

func pullOpenstack(me, cmd, name string) error {

	showCredentialsOpenstack()

	regionName := os.Getenv("OS_REGION_NAME")
	if regionName == "" {
		return fmt.Errorf("missing env var OS_REGION_NAME")
	}

	opts, errAuth := openstack.AuthOptionsFromEnv()
	if errAuth != nil {
		return errAuth
	}

	provider, errProv := openstack.AuthenticatedClient(opts)
	if errProv != nil {
		return errProv
	}

	client, errClient := openstack.NewNetworkV2(provider, gophercloud.EndpointOpts{
		Region: regionName,
	})
	if errClient != nil {
		return errClient
	}

	groupID, errID := groups.IDFromName(client, name)
	if errID != nil {
		return errID
	}

	sg, errGet := groups.Get(client, groupID).Extract()
	if errGet != nil {
		return errGet
	}

	gr := group{
		Description: sg.Description,
	}

	for _, sgr := range sg.Rules {
		var r rule

		r.PortFirst = int64(sgr.PortRangeMin)
		r.PortLast = int64(sgr.PortRangeMax)
		r.Protocol = sgr.Protocol

		if sgr.RemoteGroupID != "" {
			log.Printf("unsupported: RemoteGroupID=[%s]", sgr.RemoteGroupID)

			if sgr.RemoteIPPrefix == "" {
				continue // do not install default prefix by accident
			}
		}

		visitDstPrefix(&r, sgr.RemoteIPPrefix, "")

		if sgr.Direction == "ingress" {
			gr.RulesIn = append(gr.RulesIn, r)
		} else {
			gr.RulesOut = append(gr.RulesOut, r)
		}
	}

	buf, errDump := gr.Dump()
	if errDump != nil {
		return errDump
	}
	fmt.Printf(string(buf))

	return nil
}
