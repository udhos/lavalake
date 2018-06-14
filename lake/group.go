package main

import (
	"gopkg.in/yaml.v2"
)

type group struct {
	AwsDescription string // !azure
	RulesIn        []rule
	RulesOut       []rule
}

type rule struct {
	AzurePriority              int32    // !aws
	AzureName                  string   // !aws
	AzureDeny                  bool     // !aws
	AzureDescription           string   // !aws
	AzureSourcePortRanges      []string // !aws
	AzureSourceAddressPrefixes []string // !aws
	Protocol                   string
	PortFirst                  int64
	PortLast                   int64
	Blocks                     []block
	BlocksV6                   []block
}

type block struct {
	Address        string
	AwsDescription string // !azure
}

func (g *group) Dump() ([]byte, error) {
	return yaml.Marshal(g)
}
