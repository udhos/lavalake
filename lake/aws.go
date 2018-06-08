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

	//log.Printf("%s: aws name=%s vpc=%s", me, name, vpcID)

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

	log.Printf("security groups: %d", len(out.SecurityGroups))

	return nil
}
