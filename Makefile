# provendb-verify
# Copyright (C) 2019  Southbank Software Ltd.
# 
# This program is free software: you can redistribute it and/or modify
# it under the terms of the GNU Affero General Public License as published by
# the Free Software Foundation, either version 3 of the License, or
# (at your option) any later version.
# 
# This program is distributed in the hope that it will be useful,
# but WITHOUT ANY WARRANTY; without even the implied warranty of
# MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
# GNU Affero General Public License for more details.
# 
# You should have received a copy of the GNU Affero General Public License
# along with this program.  If not, see <http://www.gnu.org/licenses/>.
# 
# 
# @Author: guiguan
# @Date:   2018-08-01T15:46:09+10:00
# @Last modified by:   guiguan
# @Last modified time: 2020-05-20T11:50:01+10:00

PROJECT_IMPORT_PATH := github.com/SouthbankSoftware/provendb-verify
APP_NAME := provendb-verify
APP_VERSION ?= 0.0.0
BC_TOKEN ?=
PLAYGROUND_NAME := playground
PKGS := $(shell go list ./cmd/... ./pkg/...)
LD_FLAGS := -ldflags \
"-X $(PROJECT_IMPORT_PATH)/pkg/proof/anchor.bcToken=$(BC_TOKEN) \
-X main.cmdVersion=$(APP_VERSION)"

all: build

.PHONY: run build build-regen generate test test-dev clean playground doc build-all

run:
	go run $(LD_FLAGS) ./cmd/$(APP_NAME) -h
build:
	go build $(LD_FLAGS) ./cmd/$(APP_NAME)
build-regen: generate build
generate:
	go generate $(PKGS)
test:
	go test $(LD_FLAGS) $(PKGS)
test-dev:
	# -test.v verbose
	go test $(LD_FLAGS) -count=1 -test.v $(PKGS)
clean:
	go clean -testcache $(PKGS)
	rm -f $(APP_NAME)* $(PLAYGROUND_NAME)*
playground:
	go run cmd/$(PLAYGROUND_NAME)/$(PLAYGROUND_NAME).go
doc:
	godoc -http=:6060
build-all:
	go run src.techknowlogick.com/xgo --deps=https://gmplib.org/download/gmp/gmp-6.0.0a.tar.bz2 --targets=linux/amd64,windows/amd64,darwin/amd64 --pkg cmd/$(APP_NAME) $(LD_FLAGS) $(PROJECT_IMPORT_PATH)
