#!/bin/bash

gofmt -s -w ./*/*.go
go tool fix ./*/*.go
go tool vet ./lake

hash golint && golint lake

go test ./lake
go install ./lake
