#!/bin/bash
# need go 1.10 to work on Darwin
go build -buildmode=plugin -o alerting.so  alerting/alerting.go
go build -buildmode=plugin -o leet.so leet/leet.go