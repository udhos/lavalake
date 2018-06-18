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
		if len(args) < 2 {
			log.Printf("usage: %s %s %s name resource-group", me, cmd, cloud)
			return fmt.Errorf("%s %s %s: missing name resource-group", me, cmd, cloud)
		}
		name := args[0]
		resourceGroup := args[1]
		return pushAzure(me, cmd, name, resourceGroup)
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

	r.AzureSourcePortRanges = []string{unptr(prop.SourcePortRange)}
	for _, src := range *prop.SourcePortRanges {
		r.AzureSourcePortRanges = append(r.AzureSourcePortRanges, src)
	}

	r.AzureSourceAddressPrefixes = []string{unptr(prop.SourceAddressPrefix)}
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
		visitDstPrefix(&r, unptr(prop.DestinationAddressPrefix), "*")
	}
	for _, dst := range *prop.DestinationAddressPrefixes {
		visitDstPrefix(&r, dst, "*")
	}

	if prop.Direction == network.SecurityRuleDirectionInbound {
		gr.RulesIn = append(gr.RulesIn, r)
	} else {
		gr.RulesOut = append(gr.RulesOut, r)
	}
}

// expand magic prefix to both IPv6 and IPv4
func visitDstPrefix(r *rule, prefix, magicDefault string) {

	if prefix == magicDefault {
		log.Printf("replacing magicDefault='%s' with 0.0.0.0/0 and ::/0", magicDefault)
		visitDstPrefix(r, "0.0.0.0/0", magicDefault)
		visitDstPrefix(r, "::/0", magicDefault)
		return
	}

	if isPrefixV6(prefix) {
		r.BlocksV6 = append(r.BlocksV6, block{Address: prefix})
	} else {
		r.Blocks = append(r.Blocks, block{Address: prefix})
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

func pushAzure(me, cmd, name, resourceGroup string) error {

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
		return createAzure(nsgClient, name, resourceGroup, &gr)
	}

	return updateAzure(nsgClient, name, resourceGroup, &gr, unptr(sg.ID))
}

func createAzure(nsgClient network.SecurityGroupsClient, name, resourceGroup string, gr *group) error {
	return updateAzure(nsgClient, name, resourceGroup, gr, "")
}

func updateAzure(nsgClient network.SecurityGroupsClient, name, resourceGroup string, gr *group, groupID string) error {

	nsg := networkSecurityGroupFromGroup(gr)

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

func networkSecurityGroupFromGroup(gr *group) network.SecurityGroup {

	list := []network.SecurityRule{}

	format := &network.SecurityGroupPropertiesFormat{
		SecurityRules: &list,
	}

	sg := network.SecurityGroup{
		SecurityGroupPropertiesFormat: format,
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

	format := &network.SecurityRulePropertiesFormat{
		Description:           to.StringPtr(r.AzureDescription),
		Protocol:              network.SecurityRuleProtocol(r.Protocol),
		Direction:             direction,
		DestinationPortRanges: &dstPortRanges,
		SourcePortRanges:      &r.AzureSourcePortRanges,
		SourceAddressPrefixes: &r.AzureSourceAddressPrefixes,
		Priority:              to.Int32Ptr(r.AzurePriority),
	}

	if r.AzureDeny {
		format.Access = network.SecurityRuleAccessDeny
	} else {
		format.Access = network.SecurityRuleAccessAllow
	}

	sr := network.SecurityRule{
		Name: to.StringPtr(r.AzureName),
		SecurityRulePropertiesFormat: format,
	}

	for _, b := range r.Blocks {
		*format.DestinationAddressPrefixes = append(*format.DestinationAddressPrefixes, b.Address)
	}

	for _, b := range r.BlocksV6 {
		*format.DestinationAddressPrefixes = append(*format.DestinationAddressPrefixes, b.Address)
	}

	return sr
}
