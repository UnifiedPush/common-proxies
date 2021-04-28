default: local
docker:
	docker run --rm -it -v "$$PWD":/usr/src/myapp -w /usr/src/myapp golang:1.15 make all	
local:
	install -d bin
	docker run --rm -it -v "$$PWD":/usr/src/myapp -w /usr/src/myapp golang:1.15 go build -o bin/up-rewrite-linux-amd64	
all:
	./go-build-all.sh	
	install -d bin
	for f in myapp*; do \
   	    mv -- "$$f" "bin/up-rewrite$${f#myapp}"; \
	done
	cd bin; \
	sha256sum * > sha256
