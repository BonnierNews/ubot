#!/bin/bash
# need go 1.10.2 to work on Darwin
go build -buildmode=plugin -o plugins/leet.so plugins/leet/leet.go
go build -buildmode=plugin -o plugins/alerts.so plugins/alerts/alerts.go
