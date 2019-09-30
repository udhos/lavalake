package main

import (
	"fmt"
	"log"
	"os"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/security/groups"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/security/rules"
	//"github.com/gophercloud/gophercloud/openstack/utils"
)

func cloudOpenstack(me, cmd, cloud string, args []string) error {

	switch cmd {
	case "list":
		return listOpenstack(me, cmd)
	case "pull":
		if len(args) < 1 {
			log.Printf("usage: %s %s %s name", me, cmd, cloud)
			return fmt.Errorf("%s %s %s: missing name", me, cmd, cloud)
		}
		name := args[0]
		return pullOpenstack(me, cmd, name)
	case "push":
		if len(args) < 1 {
			log.Printf("usage: %s %s %s name", me, cmd, cloud)
			return fmt.Errorf("%s %s %s: missing name", me, cmd, cloud)
		}
		name := args[0]
		return pushOpenstack(me, cmd, name)
	}

	return fmt.Errorf("unsupported %s command: %s", cloud, cmd)
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

		isPrefixV6 := sgr.EtherType == "IPv6"

		visitDstPrefixV(&r, sgr.RemoteIPPrefix, "", isPrefixV6)

		if sgr.Direction == "ingress" {
			gr.RulesIn = append(gr.RulesIn, r)
		} else {
			gr.RulesOut = append(gr.RulesOut, r)
		}
	}

	gr.output()

	return nil
}

func pushOpenstack(me, cmd, name string) error {

	var gr group

	if errLoad := groupFromStdin(me, name, &gr); errLoad != nil {
		return errLoad
	}

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
		log.Printf("%s: group=%s not found: %v", me, name, errID)
		return createOpenstack(client, &gr, me, name)
	}

	return updateOpenstack(client, &gr, me, name, groupID)
}

func createOpenstack(client *gophercloud.ServiceClient, gr *group, me, name string) error {
	log.Printf("%s: creating new group=%s", me, name)

	createOpts := groups.CreateOpts{
		Name: name,
	}

	sg, errCreate := groups.Create(client, createOpts).Extract()
	if errCreate != nil {
		return errCreate
	}

	log.Printf("%s: created group=%s group-id=%s", me, name, sg.ID)

	return updateOpenstack(client, gr, me, name, sg.ID)
}

func updateOpenstack(client *gophercloud.ServiceClient, gr *group, me, name, groupID string) error {
	log.Printf("%s: updating existing group=%s group-id=%s", me, name, groupID)

	updateOpts := groups.UpdateOpts{
		Description: &gr.Description,
	}

	if _, errUpdateDesc := groups.Update(client, groupID, updateOpts).Extract(); errUpdateDesc != nil {
		return errUpdateDesc
	}

	log.Printf("%s: group=%s updated description=[%s]", me, name, gr.Description)

	sg, errGet := groups.Get(client, groupID).Extract()
	if errGet != nil {
		return errGet
	}

	log.Printf("%s: group=%s deleting %d rules...", me, name, len(sg.Rules))

	for _, sgr := range sg.Rules {
		errDel := rules.Delete(client, sgr.ID).ExtractErr()
		if errDel != nil {
			return errDel
		}
	}

	log.Printf("%s: group=%s deleting %d rules...done", me, name, len(sg.Rules))

	log.Printf("%s: group=%s creating new rules...", me, name)

	countIn, errIn := scanRulesOpenstack(client, gr.RulesIn, groupID, rules.DirIngress)
	if errIn != nil {
		return errIn
	}
	countOut, errOut := scanRulesOpenstack(client, gr.RulesOut, groupID, rules.DirEgress)
	if errOut != nil {
		return errOut
	}

	log.Printf("%s: group=%s creating new rules...done (%d rules)", me, name, countIn+countOut)

	return nil
}

func scanRulesOpenstack(client *gophercloud.ServiceClient, ruleList []rule, groupID string, direction rules.RuleDirection) (int, error) {
	var count int

	for _, r := range ruleList {
		for _, b := range r.Blocks {
			createOpts := createRuleOpenstack(r, groupID, b.Address, rules.EtherType4, direction)
			_, errCreate := rules.Create(client, createOpts).Extract()
			if errCreate != nil {
				return count, errCreate
			}
			count++
		}
		for _, b := range r.BlocksV6 {
			createOpts := createRuleOpenstack(r, groupID, b.Address, rules.EtherType6, direction)
			_, errCreate := rules.Create(client, createOpts).Extract()
			if errCreate != nil {
				return count, errCreate
			}
			count++
		}
	}

	return count, nil
}

func createRuleOpenstack(r rule, groupID, prefix string, etherType rules.RuleEtherType, direction rules.RuleDirection) rules.CreateOpts {
	createOpts := rules.CreateOpts{
		Direction:      direction,
		PortRangeMin:   int(r.PortFirst),
		PortRangeMax:   int(r.PortLast),
		EtherType:      etherType,
		Protocol:       rules.RuleProtocol(r.Protocol),
		SecGroupID:     groupID,
		RemoteIPPrefix: prefix,
	}
	return createOpts
}
