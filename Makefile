BUILD_DIR=./bin
VERSION=`git describe --tags`

# pass -e DOCKER_OUTPUT=registry to make to push images
DOCKER_OUTPUT ?= local

local:
	go build -o up-rewrite
local-docker:
	docker run --rm -it -v "$$PWD":/usr/src/myapp -w /usr/src/myapp golang:1.15 make local
all:
	install -d bin
	OUTPUT="${BUILD_DIR}/up-rewrite" ./go-build-all.sh	
	cd bin; \
		sha256sum * > sha256
all-docker:
	docker run --rm -it -v "$$PWD":/usr/src/myapp -w /usr/src/myapp golang:1.15 make all	

test: local
	go test # no tests yet defined as of writing this

# for local testing
build-docker-amd64: linux/amd64/build-docker

# for CI
build-docker-all: linux/amd64,linux/386,linux/arm64/build-docker


# check out this if the cross-docker things don't work https://stackoverflow.com/a/65371609/8919142
%/build-docker:
	cp .gitignore .dockerignore
	sed 's/127.0.0.1/0.0.0.0/' example-config.toml > config.toml # very very stopgap solution until env vars config works
	docker buildx build --platform $(@D) --output=type=$(DOCKER_OUTPUT) \
		--cache-to type=local,dest=bin/docker \
		--cache-from type=local,src=bin/docker \
		--pull \
                -t unifiedpush/common-proxies:latest \
                -t unifiedpush/common-proxies:${VERSION} \
                -t unifiedpush/common-proxies:$(shell echo $(VERSION) | cut -d '.' -f -2) .
	rm config.toml .dockerignore
