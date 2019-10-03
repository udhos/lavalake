package main

import (
	"context"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

func cloudAws(me, cmd, cloud string, args []string) error {

	switch cmd {
	case "list":
		var vpcID string
		if len(args) > 0 {
			vpcID = args[0]
		}
		return listAws(me, cmd, vpcID)
	case "pull":
		if len(args) < 2 {
			log.Printf("usage: %s %s %s name vpc-id", me, cmd, cloud)
			return fmt.Errorf("%s %s %s: missing name vpc-id", me, cmd, cloud)
		}
		name := args[0]
		vpcID := args[1]
		return pullAws(me, cmd, name, vpcID)
	case "push":
		if len(args) < 2 {
			log.Printf("usage: %s %s %s name vpc-id", me, cmd, cloud)
			return fmt.Errorf("%s %s %s: missing name vpc-id", me, cmd, cloud)
		}
		name := args[0]
		vpcID := args[1]
		return pushAws(me, cmd, name, vpcID)
	}

	return fmt.Errorf("unsupported %s command: %s", cloud, cmd)
}

func listAws(me, cmd, vpcID string) error {
	cfg, errConf := external.LoadDefaultAWSConfig()
	if errConf != nil {
		return errConf
	}

	svc := ec2.New(cfg)

	input := ec2.DescribeSecurityGroupsInput{}

	if vpcID != "" {
		filterVpc := ec2.Filter{
			Name:   aws.String("vpc-id"),
			Values: []string{vpcID},
		}
		input.Filters = []ec2.Filter{filterVpc}
	}

	req := svc.DescribeSecurityGroupsRequest(&input)

	out, errSend := req.Send(context.TODO())
	if errSend != nil {
		return errSend
	}

	count := len(out.SecurityGroups)
	log.Printf("security groups: %d", count)

	for _, sg := range out.SecurityGroups {
		fmt.Printf("vpc-id=%s group-name=%s group-id=%s description=%s\n",
			aws.StringValue(sg.VpcId), aws.StringValue(sg.GroupName), aws.StringValue(sg.GroupId), aws.StringValue(sg.Description))
	}

	return nil
}

func pullAws(me, cmd, name, vpcID string) error {
	cfg, errConf := external.LoadDefaultAWSConfig()
	if errConf != nil {
		return errConf
	}

	svc := ec2.New(cfg)

	filterName := ec2.Filter{
		Name:   aws.String("group-name"),
		Values: []string{name},
	}

	filterVpc := ec2.Filter{
		Name:   aws.String("vpc-id"),
		Values: []string{vpcID},
	}

	input := ec2.DescribeSecurityGroupsInput{
		Filters: []ec2.Filter{filterName, filterVpc},
	}

	req := svc.DescribeSecurityGroupsRequest(&input)

	out, errSend := req.Send(context.TODO())
	if errSend != nil {
		return errSend
	}

	count := len(out.SecurityGroups)
	log.Printf("security groups: %d", count)

	if count < 1 {
		return fmt.Errorf("no security group found")
	}

	if count > 1 {
		return fmt.Errorf("more than one security group found")
	}

	sg := out.SecurityGroups[0]

	gr := group{
		Description: aws.StringValue(sg.Description),
	}

	gr.RulesIn = scanPerm(name, sg.IpPermissions)
	gr.RulesOut = scanPerm(name, sg.IpPermissionsEgress)

	gr.output()

	return nil
}

func awsProtoPull(p string) string {
		if p == "-1" {
			log.Printf("awsProtoPull: replacing protocol='-1' with empty string")
			return ""
		}
	return p
}

func awsProtoPush(p string) string {
		if p == "" {
			log.Printf("awsProtoPush: replacing empty protocol with '-1'")
			return "-1"
		}
	return p
}

func scanPerm(name string, permissions []ec2.IpPermission) []rule {

	var rules []rule

	for _, perm := range permissions {
		for _, other := range perm.UserIdGroupPairs {
			log.Printf("unsupported: this group=%s references another group=%s", name, aws.StringValue(other.GroupId))
		}

		proto := awsProtoPull(aws.StringValue(perm.IpProtocol))

		r := rule{
			Protocol:  proto,
			PortFirst: aws.Int64Value(perm.FromPort),
			PortLast:  aws.Int64Value(perm.ToPort),
		}
		for _, b := range perm.IpRanges {
			blk := block{
				Address:        aws.StringValue(b.CidrIp),
				AwsDescription: aws.StringValue(b.Description),
			}
			r.Blocks = append(r.Blocks, blk)
		}
		for _, b := range perm.Ipv6Ranges {
			blk := block{
				Address:        aws.StringValue(b.CidrIpv6),
				AwsDescription: aws.StringValue(b.Description),
			}
			r.BlocksV6 = append(r.BlocksV6, blk)
		}
		rules = append(rules, r)
	}

	return rules
}

func pushAws(me, cmd, name, vpcID string) error {

	var gr group

	if errLoad := groupFromStdin(me, name, &gr); errLoad != nil {
		return errLoad
	}

	cfg, errConf := external.LoadDefaultAWSConfig()
	if errConf != nil {
		return errConf
	}

	svc := ec2.New(cfg)

	filterName := ec2.Filter{
		Name:   aws.String("group-name"),
		Values: []string{name},
	}

	filterVpc := ec2.Filter{
		Name:   aws.String("vpc-id"),
		Values: []string{vpcID},
	}

	input := ec2.DescribeSecurityGroupsInput{
		Filters: []ec2.Filter{filterName, filterVpc},
	}

	req := svc.DescribeSecurityGroupsRequest(&input)

	out, errSend := req.Send(context.TODO())
	if errSend != nil {
		return errSend
	}

	count := len(out.SecurityGroups)

	if count > 1 {
		return fmt.Errorf("more than one security group found")
	}

	if count < 1 {
		log.Printf("%s: group=%s vpc-id=%s not found", me, name, vpcID)
		return createAws(svc, &gr, name, vpcID)
	}

	sg := out.SecurityGroups[0]

	return updateAws(svc, &gr, name, vpcID, aws.StringValue(sg.GroupId))
}

func updateAws(svc *ec2.Client, gr *group, name, vpcID, groupID string) error {
	log.Printf("updating existing group=%s group-id=%s", name, groupID)

	filterName := ec2.Filter{
		Name:   aws.String("group-name"),
		Values: []string{name},
	}

	filterVpc := ec2.Filter{
		Name:   aws.String("vpc-id"),
		Values: []string{vpcID},
	}

	input := ec2.DescribeSecurityGroupsInput{
		Filters: []ec2.Filter{filterName, filterVpc},
	}

	req := svc.DescribeSecurityGroupsRequest(&input)

	out, errSend := req.Send(context.TODO())
	if errSend != nil {
		return errSend
	}

	count := len(out.SecurityGroups)

	if count < 1 {
		return fmt.Errorf("no security group found")
	}

	if count > 1 {
		return fmt.Errorf("more than one security group found")
	}

	sg := out.SecurityGroups[0]

	if groupID != aws.StringValue(sg.GroupId) {
		return fmt.Errorf("wrong groupID")
	}

	countInDel := countBlocks(sg.IpPermissions)
	log.Printf("group=%s deleting existing %d ingress rules...", name, countInDel)
	if errDelIn := delPermInAws(svc, sg); errDelIn != nil {
		return errDelIn
	}
	log.Printf("group=%s deleting existing %d ingress rules...done", name, countInDel)

	countOutDel := countBlocks(sg.IpPermissionsEgress)
	log.Printf("group=%s deleting existing %d egress rules...", name, countOutDel)
	if errDelIn := delPermOutAws(svc, sg); errDelIn != nil {
		return errDelIn
	}
	log.Printf("group=%s deleting existing %d egress rules...done", name, countOutDel)

	log.Printf("group=%s creating new rules...", name)

	countIn, errIn := addPermInAws(svc, gr.RulesIn, name, groupID)
	if errIn != nil {
		return fmt.Errorf("addPermInAws: %v", errIn)
	}

	countOut, errOut := addPermOutAws(svc, gr.RulesOut, name, groupID)
	if errOut != nil {
		return fmt.Errorf("addPermOutAws: %v", errOut)
	}

	log.Printf("group=%s creating new rules...done (%d rules)", name, countIn+countOut)

	return nil
}

func delPermInAws(svc *ec2.Client, sg ec2.SecurityGroup) error {

	if len(sg.IpPermissions) < 1 {
		return nil
	}

	input := ec2.RevokeSecurityGroupIngressInput{
		IpPermissions: sg.IpPermissions,
		GroupId:       sg.GroupId,
	}
	req := svc.RevokeSecurityGroupIngressRequest(&input)
	_, err := req.Send(context.TODO())
	return err
}

func delPermOutAws(svc *ec2.Client, sg ec2.SecurityGroup) error {

	if len(sg.IpPermissionsEgress) < 1 {
		return nil
	}

	input := ec2.RevokeSecurityGroupEgressInput{
		IpPermissions: sg.IpPermissionsEgress,
		GroupId:       sg.GroupId,
	}
	req := svc.RevokeSecurityGroupEgressRequest(&input)
	_, err := req.Send(context.TODO())
	return err
}

func countBlocks(permissions []ec2.IpPermission) int {
	var count int
	for _, perm := range permissions {
		count += len(perm.IpRanges) + len(perm.Ipv6Ranges)
	}
	return count
}

func permFromRules(ruleList []rule) ([]ec2.IpPermission, int) {
	var permissions []ec2.IpPermission
	var count int

	for _, r := range ruleList {
		if len(r.Blocks) < 1 && len(r.BlocksV6) < 1 {
			continue
		}
		proto := awsProtoPush(r.Protocol)
		perm := ec2.IpPermission{
			IpProtocol: aws.String(proto),
			FromPort:   aws.Int64(r.PortFirst),
			ToPort:     aws.Int64(r.PortLast),
		}
		for _, b := range r.Blocks {
			perm.IpRanges = append(perm.IpRanges, ec2.IpRange{
				CidrIp:      aws.String(b.Address),
				Description: aws.String(b.AwsDescription),
			})
			count++
		}
		for _, b := range r.BlocksV6 {
			perm.Ipv6Ranges = append(perm.Ipv6Ranges, ec2.Ipv6Range{
				CidrIpv6:    aws.String(b.Address),
				Description: aws.String(b.AwsDescription),
			})
			count++
		}
		permissions = append(permissions, perm)
	}

	return permissions, count
}

func addPermInAws(svc *ec2.Client, ruleList []rule, name, groupID string) (int, error) {

	permissions, count := permFromRules(ruleList)

	log.Printf("addPermInAws: count=%d", count)

	if count < 1 {
		return count, nil
	}

	input := ec2.AuthorizeSecurityGroupIngressInput{
		IpPermissions: permissions,
		GroupId:       aws.String(groupID),
	}
	req := svc.AuthorizeSecurityGroupIngressRequest(&input)
	_, err := req.Send(context.TODO())

	return count, err
}

func addPermOutAws(svc *ec2.Client, ruleList []rule, name, groupID string) (int, error) {

	permissions, count := permFromRules(ruleList)

	log.Printf("addPermOutAws: count=%d", count)

	if count < 1 {
		return count, nil
	}

	input := ec2.AuthorizeSecurityGroupEgressInput{
		IpPermissions: permissions,
		GroupId:       aws.String(groupID),
	}
	req := svc.AuthorizeSecurityGroupEgressRequest(&input)
	_, err := req.Send(context.TODO())

	return count, err
}

func createAws(svc *ec2.Client, gr *group, name, vpcID string) error {
	log.Printf("createAws: creating new group=%s vpc-id=%s", name, vpcID)

	var desc string
	if gr.Description != "" {
		desc = gr.Description
	} else {
		desc = name
		log.Printf("createAws: using group name as description: %s", desc)
	}

	input := ec2.CreateSecurityGroupInput{
		GroupName:   aws.String(name),
		VpcId:       aws.String(vpcID),
		Description: aws.String(desc),
	}

	req := svc.CreateSecurityGroupRequest(&input)
	resp, errCreate := req.Send(context.TODO())
	if errCreate != nil {
		return fmt.Errorf("createAws: %v", errCreate)
	}

	groupID := aws.StringValue(resp.GroupId)

	log.Printf("created new group=%s vpc-id=%s: group-id=%s", name, vpcID, groupID)

	return updateAws(svc, gr, name, vpcID, groupID)
}
