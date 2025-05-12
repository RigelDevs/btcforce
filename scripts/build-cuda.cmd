@echo off
set CGO_ENABLED=1
go build -v -o btcforce.exe cmd\btcforce\main.go