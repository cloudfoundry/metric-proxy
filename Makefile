# With thanks to:
# https://dev.to/eugenebabichenko/generating-pretty-version-strings-including-nightly-with-git-and-makefiles-48p3
TAG_COMMIT := $(shell git rev-list --abbrev-commit --tags --max-count=1)
TAG := $(shell git describe --abbrev=0 --tags ${TAG_COMMIT} 2>/dev/null || true)
COMMIT := $(shell git rev-parse --short HEAD)
DATE := $(shell git log -1 --format=%cd --date=format:"%Y%m%d")
VERSION := $(TAG:v%=%)

ifneq ($(COMMIT), $(TAG_COMMIT))
	VERSION := $(VERSION)-untagged-$(COMMIT)-$(DATE)
endif
ifeq ($(VERSION),)
	VERSION := $(COMMIT)-$(DATA)
endif
ifneq ($(shell git status --porcelain),)
	VERSION := $(VERSION)-dirty
endif

default: build-linux

build-all: build-linux build-windows build-darwin

build-linux:
	echo "Building linux binary tag $(VERSION) in bin/metric-proxy-linux..."
	GOOS=linux GOARCH=amd64 go build -a -installsuffix nocgo -o bin/metric-proxy-linux -ldflags "-X main.version=$(VERSION)" -mod=readonly .

build-windows:
	echo "Building windows binary tag $(VERSION) in bin/metric-proxy-windows..."
	GOOS=windows GOARCH=amd64 go build -a -installsuffix nocgo -o bin/metric-proxy-windows -ldflags "-X main.version=$(VERSION)" -mod=readonly .

build-darwin:
	echo "Building darwin binary tag $(VERSION) in bin/metric-proxy-darwin..."
	GOOS=darwin OARCH=amd64 go build -a -installsuffix nocgo -o bin/metric-proxy-darwin -ldflags "-X main.version=$(VERSION)" -mod=readonly .

test:
	go test ./... -race -count=1

clean:
	rm bin/*
