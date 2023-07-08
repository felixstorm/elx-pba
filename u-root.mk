.build/u-root/.git/HEAD:
	rm -rf .build/u-root 2>/dev/null
	mkdir -p .build/u-root
	git clone https://github.com/u-root/u-root .build/u-root
	(cd .build/u-root; git reset $(UROOT_GIT_REF) --hard)

.build/u-root/u-root: .build/u-root/.git/HEAD
	(cd .build/u-root; GOPATH="$(GOPATH)" go build)
