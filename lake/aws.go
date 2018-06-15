package main

import (
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
		return fmt.Errorf("%s FIXME WRITEME: cmd=%s", cloud, cmd)
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

	out, errSend := req.Send()
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

	out, errSend := req.Send()
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

	buf, errDump := gr.Dump()
	if errDump != nil {
		return errDump
	}
	fmt.Printf(string(buf))

	return nil
}

func scanPerm(name string, permissions []ec2.IpPermission) []rule {

	var rules []rule

	for _, perm := range permissions {
		for _, other := range perm.UserIdGroupPairs {
			log.Printf("unsupported: this group=%s references another group=%s", name, aws.StringValue(other.GroupId))
		}

		proto := aws.StringValue(perm.IpProtocol)
		if proto == "-1" {
			log.Printf("replacing protocol='-1' with empty string")
			proto = ""
		}

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
