#
# Makefile for Building the Go Binary
# Building an ARM7 Binary for Raspberry Pi requires docker
#

OS 			?= $(shell echo `uname -s` | awk '{print tolower($0)}')
ARCH 		?= $(shell echo `uname -m` | awk '{print tolower($0)}')
CGO_ENABLED ?= 1
CGO_CFLAGS 	?= ""
CGO_LDFLAGS ?= ""
GOOS 		?=
GOARCH 		?=
GOARM 		?=
GOOUT  		?= "./sfmplayer.$(OS)-$(ARCH)"

.PHONY: build

build:
	GOOS=$(GOOS) \
	GOARCH=$(GOARCH) \
	GOARM=$(GOARM) \
	CGO_ENABLED=$(CGO_ENABLED) \
	CGO_LDFLAGS=$(CGO_LDFLAGS) \
	CGO_CFLAGS=$(CGO_CFLAGS) \
	go build -v -o $(GOOUT)
