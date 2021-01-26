docker:
	docker run --rm -v "$PWD":/usr/src/myapp -w /usr/src/myapp golang:1.15 make all	
all:
	./go-build-all.sh	
	install -d bin
	for f in myapp*; do \
   	    mv -- "$$f" "bin/up-rewrite$${f#myapp}"; \
	done
	cd bin; \
	sha256sum * > sha256
