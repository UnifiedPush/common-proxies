local:
	go build -o up-rewrite
local-docker:
	docker run --rm -it -v "$$PWD":/usr/src/myapp -w /usr/src/myapp golang:1.15 make local
all:
	install -d bin
	OUTPUT="bin/up-rewrite" ./go-build-all.sh	
	cd bin; \
		sha256sum * > sha256
all-docker:
	docker run --rm -it -v "$$PWD":/usr/src/myapp -w /usr/src/myapp golang:1.15 make all	

test: local
	go test # no tests yet defined as of writing this
