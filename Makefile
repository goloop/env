# GOLOOP
#
# Load data about the package:
# 	REPOSITORY_NAME
# 		The name of the repository where the package is stored,
# 		for example: github.com/goloop;
#	MODULE_NAME
#		Name of the GoLang's pakcage.
#
# The REPOSITORY_NAME and MODULE_NAME create a path to import the package
# (save the source code in GOPATH as $GOPATH/src/REPOSITORY_NAME/MODULE_NAME)
SRC_PATH:=$(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
include $(SRC_PATH)/.module

ifndef MODULE_NAME
	$(error Add PAKCAGE_NAME in .package)
endif

ifndef REPOSITORY_NAME
	$(error Add REPOSITORY_NAME in .package)
endif

# Help information.
define MSG_HELP
Go-package manager.

Commands:
	help
		Show this help information
	go.test
		Run tests
	go.test.cover
		Check test coverage
	go.lint
		Check cod with GoLints
		 
		Requires `golangci-lint`, install as:
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.24.0
	readme
		Create readme from the GoLang code
		
		Requires `godocdown`, install as:
		go get github.com/robertkrimen/godocdown/godocdown
endef

# Constants.
export MSG_HELP
REPOSITORY_PATH=${REPOSITORY_NAME}/${MODULE_NAME}

all: help
help:
	@echo "$$MSG_HELP"
go.test:
	@go clean -testcache; go test ${REPOSITORY_PATH}
go.test.cover:
	@go test -cover ${REPOSITORY_PATH} && \
		go test -coverprofile=/tmp/coverage.out ${REPOSITORY_PATH} && \
		go tool cover -func=/tmp/coverage.out && \
		go tool cover -html=/tmp/coverage.out
go.lint:
ifeq (, $(shell which golangci-lint))
	@curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.24.0
endif
	golangci-lint run --no-config --issues-exit-code=0 --timeout=30m \
		--disable-all --enable=deadcode  --enable=gocyclo --enable=golint \
		--enable=varcheck --enable=structcheck --enable=maligned \
		--enable=errcheck --enable=dupl --enable=ineffassign \
		--enable=interfacer --enable=unconvert --enable=goconst \
		--enable=gosec --enable=megacheck
readme:
ifeq (, $(shell which godocdown))
	@go get github.com/robertkrimen/godocdown/godocdown
endif
	@godocdown -plain=true -template=.godocdown.md ./ | \
		sed -e 's/\.VERSION/${VERSION}/g' > README.md
