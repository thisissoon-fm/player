#
# Makefile for Building the Go Binary
# Building an ARM7 Binary for Raspberry Pi requires docker
#

OS 					?= $(shell echo `uname -s` | awk '{print tolower($0)}')
ARCH 				?= $(shell echo `uname -m` | awk '{print tolower($0)}')
CGO_ENABLED 		?= 1
CGO_CFLAGS 			?= ""
CGO_LDFLAGS 		?= ""
GOOS 				?=
GOARCH 				?=
GOARM 				?=
GOOUTDIR 			?= .
GOOUT  				?= "$(GOOUTDIR)/sfmplayer.$(OS)-$(ARCH)"
BUILD_TIME 			?= $(shell date +%s)
BUILD_VERSION 		?= $(shell git rev-parse --short HEAD)
BUILD_TIME_FLAG 	?= -X player/build.timestamp=${BUILD_TIME}
BUILD_VERSION_FLAG 	?= -X player/build.version=${BUILD_VERSION}
BUILD_ARCH_FLAG 	?= -X player/build.arch=$(if $(call check_defined, GOARCH),$(GOARCH),$(ARCH))
BUILD_OS_FLAG 		?= -X player/build.os=$(if $(call check_defined, GOOS),$(GOOS),$(OS))

.PHONY: build

build:
	GODEBUG=cgocheck=0 \
	GOOS=$(GOOS) \
	GOARCH=$(GOARCH) \
	GOARM=$(GOARM) \
	CGO_ENABLED=$(CGO_ENABLED) \
	CGO_LDFLAGS=$(CGO_LDFLAGS) \
	CGO_CFLAGS=$(CGO_CFLAGS) \
	go build -v \
		-ldflags "$(BUILD_TIME_FLAG) $(BUILD_VERSION_FLAG) $(BUILD_ARCH_FLAG) $(BUILD_OS_FLAG)" \
		-o $(GOOUT)
