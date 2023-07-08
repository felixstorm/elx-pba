ifeq ($(ARCH),x86_64)
GOARCH := amd64
endif

.build/rootfs-$(ARCH).cpio: .build/u-root/u-root $(wildcard cmd/*/*.go)
	GOPATH="$(GOPATH)" go install \
		github.com/open-source-firmware/go-tcg-storage/cmd/sedlockctl@$(GO_TCG_STORAGE_GIT_REF) \
		github.com/open-source-firmware/go-tcg-storage/cmd/tcgdiskstat@$(GO_TCG_STORAGE_GIT_REF) \
		github.com/open-source-firmware/go-tcg-storage/cmd/tcgsdiag@$(GO_TCG_STORAGE_GIT_REF)
	sed -i 's|GitInfo = "[^"]*"|GitInfo = "$(LOCAL_GIT_INFO)"|g' pbainit/gitinfo.go
	(cd .build/u-root; GOPATH="$(GOPATH)" ./u-root \
		-o "$(PWD)/$(@)" \
		-build=gbb \
		-initcmd pbainit \
		-files $(GOPATH)/bin/sedlockctl:usr/local/bin/sedlockctl \
		-files $(GOPATH)/bin/tcgdiskstat:usr/local/bin/tcgdiskstat \
		-files $(GOPATH)/bin/tcgsdiag:usr/local/bin/tcgsdiag \
		core \
		boot \
		"$(PWD)/pbainit" \
		cmds/exp/partprobe \
	)
