BIN := danse
DIR := dist
GOBIN := go

LAST_COMMIT := $(shell git rev-parse --short HEAD)
LAST_COMMIT_DATE := $(shell git show -s --format=%ci ${LAST_COMMIT})
VERSION := $(shell git describe --tags)
BUILDSTR := ${VERSION} (Commit: ${LAST_COMMIT_DATE} (${LAST_COMMIT}), Build: $(shell date +"%Y-%m-%d% %H:%M:%S %z"))

.PHONY: build
build:
	mkdir -p ${DIR}
	CGO_ENABLED=0 ${GOBIN} build -o ${DIR}/${BIN} --ldflags="-X 'main.buildString=${BUILDSTR}'"
	cp ${DIR}/${BIN} .

run: build
	./${DIR}/${BIN}