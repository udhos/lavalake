package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-04-01/network"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/Azure/go-autorest/autorest/to"
)

func cloudAzure(me, cmd, cloud string, args []string) error {

	switch cmd {
	case "list":
		return listAzure(me, cmd)
	case "pull":
		if len(args) < 2 {
			log.Printf("usage: %s %s %s name resource-group", me, cmd, cloud)
			return fmt.Errorf("%s %s %s: missing name resource-group", me, cmd, cloud)
		}
		name := args[0]
		resourceGroup := args[1]
		return pullAzure(me, cmd, name, resourceGroup)
	case "push":
		if len(args) < 3 {
			log.Printf("usage: %s %s %s name resource-group location", me, cmd, cloud)
			return fmt.Errorf("%s %s %s: missing name resource-group location", me, cmd, cloud)
		}
		name := args[0]
		resourceGroup := args[1]
		location := args[2]
		return pushAzure(me, cmd, name, resourceGroup, location)
	}

	return fmt.Errorf("unsupported %s command: %s", cloud, cmd)
}

func showCredentialsAzure() {
	cred("AZURE_SUBSCRIPTION_ID")
	cred("AZURE_TENANT_ID")
	cred("AZURE_CLIENT_ID")
	credHide("AZURE_CLIENT_SECRET")
}

func cred(env string) {
	value := os.Getenv(env)
	log.Printf("credentials %s=[%s]", env, value)
}

func credHide(env string) {
	value := os.Getenv(env)
	if value != "" {
		value = "<hidden>"
	}
	log.Printf("credentials %s=[%s]", env, value)
}

func listAzure(me, cmd string) error {

	showCredentialsAzure()

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
		fmt.Printf("name=%s resource-group=%s location=%s\n", unptr(nsg.Name), unptr(nsg.SecurityGroupPropertiesFormat.ResourceGUID), unptr(nsg.Location))
	}

	return nil
}

func unptr(p *string) string {
	if p == nil {
		return "<nil-string>"
	}
	return *p
}

func unptrInt32(p *int32) int32 {
	if p == nil {
		return 0
	}
	return *p
}

func pullAzure(me, cmd, name, resourceGroup string) error {

	showCredentialsAzure()

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

	sg, errGet := nsgClient.Get(context.Background(), resourceGroup, name, "")
	if errGet != nil {
		return errGet
	}

	var gr group

	for _, sr := range *sg.SecurityGroupPropertiesFormat.SecurityRules {

		prop := sr.SecurityRulePropertiesFormat

		if nil != prop.DestinationPortRange {
			visitDstPortRange(&gr, sr, unptr(prop.DestinationPortRange))
		}
		for _, dstPortRange := range *prop.DestinationPortRanges {
			visitDstPortRange(&gr, sr, dstPortRange)
		}
	}

	gr.output()

	return nil
}

func portValue(port string) int64 {
	p, err := strconv.Atoi(port)
	if err != nil {
		log.Printf("bad port value: '%s' error: %v", port, err)
	}
	return int64(p)
}

func visitDstPortRange(gr *group, sr network.SecurityRule, dstPortRange string) {
	var r rule
	r.AzureName = unptr(sr.Name)

	prop := sr.SecurityRulePropertiesFormat

	r.Protocol = string(prop.Protocol)
	if r.Protocol == "*" {
		log.Printf("replacing protocol='*' with empty string")
		r.Protocol = ""
	}

	var desc string
	if nil != prop.Description {
		desc = unptr(prop.Description)
	}

	r.AzureDescription = desc
	r.AzurePriority = unptrInt32(prop.Priority)
	r.AzureDeny = prop.Access == network.SecurityRuleAccessDeny

	log.Printf("SourcePortRange = %v", unptr(prop.SourcePortRange))
	log.Printf("SourcePortRanges = %v", *prop.SourcePortRanges)

	r.AzureSourcePortRange = unptr(prop.SourcePortRange)

	for _, src := range *prop.SourcePortRanges {
		r.AzureSourcePortRanges = append(r.AzureSourcePortRanges, src)
	}

	log.Printf("SourceAddressPrefix = %v", unptr(prop.SourceAddressPrefix))
	log.Printf("SourceAddressPrefixes = %v", *prop.SourceAddressPrefixes)

	r.AzureSourceAddressPrefix = unptr(prop.SourceAddressPrefix)

	for _, src := range *prop.SourceAddressPrefixes {
		r.AzureSourceAddressPrefixes = append(r.AzureSourceAddressPrefixes, src)
	}

	ports := strings.Split(dstPortRange, "-")
	if len(ports) < 2 {
		r.PortFirst = portValue(ports[0])
		r.PortLast = r.PortFirst
	} else {
		r.PortFirst = portValue(ports[0])
		r.PortLast = portValue(ports[1])
	}

	if nil != prop.DestinationAddressPrefix {
		visitDstPrefix(&r, unptr(prop.DestinationAddressPrefix), "*", true)
	}
	for _, dst := range *prop.DestinationAddressPrefixes {
		visitDstPrefix(&r, dst, "*", false)
	}

	if prop.Direction == network.SecurityRuleDirectionInbound {
		gr.RulesIn = append(gr.RulesIn, r)
	} else {
		gr.RulesOut = append(gr.RulesOut, r)
	}
}

// expand magic prefix to both IPv6 and IPv4
func visitDstPrefix(r *rule, prefix, magicDefault string, azureSingle bool) {

	if prefix == magicDefault {
		log.Printf("visitDstPrefix: replacing magicDefault='%s' with 0.0.0.0/0 and ::/0", magicDefault)
		prefixAdd(r, "0.0.0.0/0", "*", "<skip>", azureSingle)
		prefixAdd(r, "::/0", "*", "<skip>", azureSingle)
		return
	}

	prefixAdd(r, prefix, "", "", azureSingle)
}

func prefixAdd(r *rule, prefix string, azurePush4, azurePush6 string, azureSingle bool) {
	if isPrefixV6(prefix) {
		r.BlocksV6 = append(r.BlocksV6, block{Address: prefix, AzurePush: azurePush6, AzureSingle: azureSingle})
	} else {
		r.Blocks = append(r.Blocks, block{Address: prefix, AzurePush: azurePush4, AzureSingle: azureSingle})
	}
}

// expand magic prefix to IPv6 or IPv4
func visitDstPrefixV(r *rule, prefix, magicDefault string, isV6 bool) {

	if isV6 {
		// IPv6
		if prefix == magicDefault {
			log.Printf("replacing magicDefault='%s' with ::/0", magicDefault)
			visitDstPrefixV(r, "::/0", magicDefault, isV6)
			return
		}

		r.BlocksV6 = append(r.BlocksV6, block{Address: prefix})

		return
	}

	// IPv4

	if prefix == magicDefault {
		log.Printf("replacing magicDefault='%s' with 0.0.0.0/0", magicDefault)
		visitDstPrefixV(r, "0.0.0.0/0", magicDefault, isV6)
		return
	}

	r.Blocks = append(r.Blocks, block{Address: prefix})
}

func isPrefixV6(prefix string) bool {
	addr, _, err := net.ParseCIDR(prefix)
	if err != nil {
		log.Printf("bad CIDR address: '%s' error: %v", prefix, err)
		return false
	}
	return addr.To4() == nil
}

func pushAzure(me, cmd, name, resourceGroup, location string) error {

	var gr group

	if errLoad := groupFromStdin(me, name, &gr); errLoad != nil {
		return errLoad
	}

	showCredentialsAzure()

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

	sg, errGet := nsgClient.Get(context.Background(), resourceGroup, name, "")
	if errGet != nil {
		log.Printf("group=%s not found: %v", name, errGet)
		return createAzure(nsgClient, name, resourceGroup, &gr, location)
	}

	return updateAzure(nsgClient, name, resourceGroup, &gr, unptr(sg.ID), location)
}

func createAzure(nsgClient network.SecurityGroupsClient, name, resourceGroup string, gr *group, location string) error {
	return updateAzure(nsgClient, name, resourceGroup, gr, "", location)
}

func updateAzure(nsgClient network.SecurityGroupsClient, name, resourceGroup string, gr *group, groupID, location string) error {

	nsg := networkSecurityGroupFromGroup(gr, location)

	nsg.ID = to.StringPtr(groupID)
	nsg.Name = to.StringPtr(name)

	future, errUpdate := nsgClient.CreateOrUpdate(context.Background(), resourceGroup, name, nsg)
	if errUpdate != nil {
		return errUpdate
	}

	_, errResult := future.Result(nsgClient)
	if errResult != nil {
		return errResult
	}

	return nil
}

func networkSecurityGroupFromGroup(gr *group, location string) network.SecurityGroup {

	list := []network.SecurityRule{}

	format := &network.SecurityGroupPropertiesFormat{
		SecurityRules: &list,
	}

	sg := network.SecurityGroup{
		SecurityGroupPropertiesFormat: format,
		Location:                      to.StringPtr(location),
	}

	for _, r := range gr.RulesIn {
		sr := securityRuleFromRule(r, network.SecurityRuleDirectionInbound)
		list = append(list, sr)
	}

	for _, r := range gr.RulesOut {
		sr := securityRuleFromRule(r, network.SecurityRuleDirectionOutbound)
		list = append(list, sr)
	}

	return sg
}

func securityRuleFromRule(r rule, direction network.SecurityRuleDirection) network.SecurityRule {

	dstPortRanges := []string{fmt.Sprintf("%d-%d", r.PortFirst, r.PortLast)}

	dstPrefixes := []string{}
	var dstPrefixSingle string

	format := &network.SecurityRulePropertiesFormat{
		Description:                to.StringPtr(r.AzureDescription),
		Protocol:                   network.SecurityRuleProtocol(r.Protocol),
		Direction:                  direction,
		DestinationPortRanges:      &dstPortRanges,
		SourcePortRange:            to.StringPtr(r.AzureSourcePortRange),
		SourcePortRanges:           &r.AzureSourcePortRanges,
		SourceAddressPrefix:        to.StringPtr(r.AzureSourceAddressPrefix),
		SourceAddressPrefixes:      &r.AzureSourceAddressPrefixes,
		Priority:                   to.Int32Ptr(r.AzurePriority),
		DestinationAddressPrefix:   &dstPrefixSingle,
		DestinationAddressPrefixes: &dstPrefixes,
	}

	if r.AzureDeny {
		format.Access = network.SecurityRuleAccessDeny
	} else {
		format.Access = network.SecurityRuleAccessAllow
	}

	sr := network.SecurityRule{
		Name:                         to.StringPtr(r.AzureName),
		SecurityRulePropertiesFormat: format,
	}

	getDstPrefixesAzure(&dstPrefixSingle, &dstPrefixes, r.Blocks)
	getDstPrefixesAzure(&dstPrefixSingle, &dstPrefixes, r.BlocksV6)

	log.Printf("DestinationAddressPrefix = %v", *sr.SecurityRulePropertiesFormat.DestinationAddressPrefix)
	log.Printf("DestinationAddressPrefixes = %v", *sr.SecurityRulePropertiesFormat.DestinationAddressPrefixes)

	return sr
}

func getDstPrefixesAzure(dstPrefixSingle *string, dstPrefixes *[]string, blocks []block) {
	for _, b := range blocks {
		address := b.Address
		log.Printf("getDstPrefixesAzure: address=%s azurePush=[%s]", address, b.AzurePush)
		if b.AzurePush == "<skip>" {
			//log.Printf("dst address=%s azurePush=[%s] - skipping address", address, b.AzurePush)
			continue // don't push
		}
		if b.AzurePush != "" {
			//log.Printf("dst address=%s azurePush=[%s] - using azurePush", address, b.AzurePush)
			address = b.AzurePush
		}
		log.Printf("getDstPrefixesAzure: azureSingle=%v address=%s", b.AzureSingle, address)
		if b.AzureSingle {
			*dstPrefixSingle = address
		} else {
			*dstPrefixes = append(*dstPrefixes, address)
		}
	}
}
