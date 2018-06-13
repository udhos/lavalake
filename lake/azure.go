package main

import (
	"fmt"
	//"log"
	"context"
	"os"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-04-01/network"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	//"github.com/Azure/go-autorest/autorest/to"
)

func cloudAzure(me, cmd string, args []string) error {

	switch cmd {
	case "list":
		return listAzure(me, cmd)
	case "pull":
		return fmt.Errorf("azure FIXME WRITEME: cmd=%s", cmd)
	case "push":
		return fmt.Errorf("azure FIXME WRITEME: cmd=%s", cmd)
	}

	return fmt.Errorf("unsupported azure command: %s", cmd)
}

func listAzure(me, cmd string) error {

	subscription := os.Getenv("AZURE_SUBSCRIPTION_ID")
	if subscription == "" {
		return fmt.Errorf("missing env var AZURE_SUBSCRIPTION_ID")
	}

	nsgClient := network.NewSecurityGroupsClient(subscription)
	authorizer, errAuth := auth.NewAuthorizerFromEnvironment()

	if errAuth != nil {
		return errAuth
	}

	nsgClient.Authorizer = authorizer

	it, errList := nsgClient.ListAllComplete(context.Background())
	if errList != nil {
		return errList
	}

	for ; it.NotDone(); it.Next() {
		nsg := it.Value()
		fmt.Printf("name=%s type=%s location=%s\n", unptr(nsg.Name), unptr(nsg.Type), unptr(nsg.Location))
	}

	return nil
}

func unptr(p *string) string {
	if p == nil {
		return "<nil-string>"
	}
	return *p
}
