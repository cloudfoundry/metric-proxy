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
	VERSION := $(COMMIT)-$(DATE)
endif
ifneq ($(shell git status --porcelain),)
	VERSION := $(VERSION)-dirty
endif

default: build-linux

all: build-linux build-windows build-darwin

build-%:
	echo "Building $* binary tag $(VERSION) in bin/metric-proxy-$*..."
	GOOS=$* GOARCH=amd64 go build -a -installsuffix nocgo -o bin/metric-proxy-$* -ldflags "-X main.version=$(VERSION)" -mod=readonly .

test:
	go test ./... -race -count=1

clean:
	rm bin/*
