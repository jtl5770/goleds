#!/bin/sh
# Build script for the Raspberry Pi

env GOOS=linux GOARCH=arm GOARM=7 go build -o goleds_pi
