BUILD_DIR=./bin
DOCKER_DIR=./docker
VERSION=`git describe --tags`

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

docker: build-docker-amd64

build-docker-amd64:
	cp ${BUILD_DIR}/up-rewrite-linux-amd64 ${DOCKER_DIR}/up-rewrite
	cp example-config.toml ${DOCKER_DIR}/config.toml
	cd ${DOCKER_DIR} && \
                docker build \
                -t unifiedpush/common-proxies:latest \
                -t unifiedpush/common-proxies:${VERSION} \
                -t unifiedpush/common-proxies:$(shell echo $(VERSION) | cut -d '.' -f -2) .
	rm ${DOCKER_DIR}/up-rewrite ${DOCKER_DIR}/config.toml
