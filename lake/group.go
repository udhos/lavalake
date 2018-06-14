package main

import (
	"gopkg.in/yaml.v2"
)

type group struct {
	Description string
	RulesIn     []rule
	RulesOut    []rule
}

type rule struct {
	AzureSequence int
	Protocol      string
	PortFirst     int64
	PortLast      int64
	Blocks        []block
	BlocksV6      []block
}

type block struct {
	Address     string
	Description string
}

func (g *group) Dump() ([]byte, error) {
	return yaml.Marshal(g)
}
