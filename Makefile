# GOLOOP
#
# Load data about the package:
#	NAME
#		Name of the GoLang's module;
#	VERSION
#		Current version.
# 	REPOSITORY
# 		The name of the repository where the package is stored,
# 		for example: github.com/goloop;
MODULE_NAME:=$(shell cat go.mod | grep module | awk '{split($$2,v,"/"); print v[3]}')
MODULE_VERSION:=$(shell cat doc.go | grep "const version" | awk '{gsub(/"/, "", $$4); print $$4}')
MODULE_MAJOR_VERSION:=$(shell cat doc.go | grep "const version" | awk '{gsub(/"/, "", $$4); print $$4}' | awk '{split($$0,r,"."); print r[1]}')
REPOSITORY_NAME:=$(shell cat go.mod | grep module | awk '{split($$2,v,"/"); print v[1] "/" v[2]}')
 
# Help information.
define MSG_HELP
Go-package's manager of
${MODULE_NAME} v${MODULE_VERSION}

Commands:
	help
		Show this help information
	test
		Run tests
	cover
		Check test coverage
	readme
		Create readme from the GoLang code
		
		Requires `godocdown`, install as:
		go get github.com/robertkrimen/godocdown/godocdown
	tag
		Create git-tag with current version of package
endef

# Constants.
export MSG_HELP
REPOSITORY_PATH=${REPOSITORY_NAME}/${MODULE_NAME}

all: help
help:
	@echo "$$MSG_HELP"
test:
	@go clean -testcache; \
	go test ${REPOSITORY_PATH}
cover:
	@go test -cover ${REPOSITORY_PATH} && \
		go test -coverprofile=/tmp/coverage.out ${REPOSITORY_PATH} && \
		go tool cover -func=/tmp/coverage.out && \
		go tool cover -html=/tmp/coverage.out
readme:
ifeq (, $(shell which godocdown))
	@go get github.com/robertkrimen/godocdown/godocdown
endif
	@godocdown -plain=true -template=.godocdown.md ./ | \
		sed -e 's/\.ModuleVersion/v${MODULE_VERSION}/g' > README.md
tag:
	@bash -c 'read -p "Do you want to create v${MODULE_VERSION} tag [y/n]?: " yn; case $${yn} in "y") git tag "v${MODULE_VERSION}"; exit 0;; *) exit 0;; esac'
