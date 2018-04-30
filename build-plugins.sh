#!/bin/bash
# need go 1.10 to work on Darwin
go build -buildmode=plugin -o plugins/leet.so plugins/leet/leet.go
