package main

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

func cloudAws(me, cmd, name string, args []string) error {

	if len(args) < 1 {
		log.Printf("usage: %s %s aws name vpc-id", me, cmd)
		return fmt.Errorf("missing vpc-id")
	}

	vpcID := args[0]

	switch cmd {
	case "pull":
		return pullAws(me, cmd, name, vpcID)
	case "push":
		return fmt.Errorf("aws FIXME WRITEME: cmd=%s", cmd)
	}

	return fmt.Errorf("unsupported aws command: %s", cmd)
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

	return fmt.Errorf("FIXME WRITEME handle security group")
}
