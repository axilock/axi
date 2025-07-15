VERSION_DEV := $(shell git describe --tags)
VERSION_LOCAL := $(shell git describe --tags --dirty)
VERSION := $(shell git describe --tags --abbrev=0)
OUT_DIR := bin

include config.mk


CONFIG = github.com/axilock/axi/internal/config
BASE_LDFLAGS = -X ${CONFIG}.version=$(VERSION) \
			   -X ${CONFIG}.debug=$(DEBUG) \
			   -X ${CONFIG}.autoupdate=$(AUTO_UPDATE) \
			   -X ${CONFIG}.grpcServerName=$(GRPC_SERVER_NAME) \
			   -X ${CONFIG}.grpcPort=$(GRPC_PORT) \
			   -X ${CONFIG}.grpcTls=$(GRPC_TLS) \
			   -X ${CONFIG}.sentryDsn=$(SENTRY_DSN) \
			   -X ${CONFIG}.backendUrl=$(BACKEND_URL)

# Run make dev DEBUG=true for debug builds

release:
	@echo "Build version = $(VERSION)"
	go build -ldflags "$(BASE_LDFLAGS) -X main.env=release" -o ${OUT_DIR}/ .

dev:
	@echo "Build version = $(VERSION_DEV)"
	go build -ldflags "$(BASE_LDFLAGS) -X main.env=dev" -o ${OUT_DIR}/ .

local:
	@echo "Build version = $(VERSION_LOCAL)"
	go build -ldflags "$(BASE_LDFLAGS) -X main.env=local" -o ${OUT_DIR}/ .

dist: dist-dev dist-release dist-list

dist-dev:
	gh workflow run 138756169

dist-release:
	gh workflow run 136795013

dist-list:
	gh run list


buildAndInstall:
	@echo "Build version = $(VERSION)"
	go build -ldflags "$(BASE_LDFLAGS) -X main.env=dev -X main.autoupdate=false" -o ${OUT_DIR}/ .
	${OUT_DIR}/axi reinstall --skip-dependencies --no-autoupdate


protos:
	 git submodule update --init --recursive    

version:
	@echo $(VERSION)

version_dev:
	@echo $(VERSION_DEV)

clean:
	rm -rf ${OUT_DIR}/*

.PHONY: axi clean
