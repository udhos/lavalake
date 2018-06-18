package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"

	"gopkg.in/yaml.v2"
)

type group struct {
	Description string // !azure
	RulesIn     []rule
	RulesOut    []rule
}

type rule struct {
	AzurePriority              int32    // azure-only
	AzureName                  string   // azure-only
	AzureDeny                  bool     // azure-only
	AzureDescription           string   // azure-only
	AzureSourcePortRange       string   // azure-only
	AzureSourcePortRanges      []string // azure-only
	AzureSourceAddressPrefix   string   // azure-only
	AzureSourceAddressPrefixes []string // azure-only
	Protocol                   string
	PortFirst                  int64
	PortLast                   int64
	Blocks                     []block
	BlocksV6                   []block
}

type block struct {
	Address        string
	AwsDescription string // aws-only
	AzurePush      string // azure-only
	AzureSingle    bool   // azure-only
}

func groupFromStdin(caller, name string, gr *group) error {
	dec := yaml.NewDecoder(bufio.NewReader(os.Stdin))

	log.Printf("%s: reading group=%s YAML from stdin...", caller, name)

	errDec := dec.Decode(gr)
	if errDec != nil && errDec != io.EOF {
		return errDec
	}

	log.Printf("%s: reading group=%s YAML from stdin...done", caller, name)

	return nil
}

func (g *group) output() {
	buf, errDump := yaml.Marshal(g)
	if errDump != nil {
		log.Printf("output: %v", errDump)
	}
	fmt.Printf(string(buf))
}
