all:
	./go-build-all.sh	
	install -d bin
	for f in rewrite_proxy*; do \
   	    mv -- "$$f" "bin/up-rewrite$${f#rewrite_proxy}"; \
	done
	cd bin; \
	sha256sum * > sha256
