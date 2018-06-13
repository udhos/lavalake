#!/bin/bash

go get gopkg.in/yaml.v2
go get github.com/go-ini/ini
go get github.com/jmespath/go-jmespath
go get github.com/aws/aws-sdk-go-v2
go get github.com/subosito/gotenv
go get -d github.com/Azure/azure-sdk-for-go/...

gofmt -s -w ./lake
go tool fix ./lake
go tool vet ./lake

hash golint && golint ./lake

go test ./lake
go install ./lake
