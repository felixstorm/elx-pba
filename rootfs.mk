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
		-files $(PWD)/arch/$(ARCH)/sedutil/sedutil-cli:usr/local/bin/sedutil-cli \
		-files $(PWD)/arch/$(ARCH)/sedutil/sedutil-cli-ca:usr/local/bin/sedutil-cli-ca \
		-files $(PWD)/arch/$(ARCH)/sedutil/sedutil_Disable_Locking.txt:sedutil_Disable_Locking.txt \
		-files $(PWD)/arch/$(ARCH)/sedutil/sedutil_unlock.txt:sedutil_unlock.txt \
		-files $(PWD)/sed_lock.sh:usr/local/bin/sed_lock.sh \
		-files $(PWD)/sed_unlock.sh:usr/local/bin/sed_unlock.sh \
		-files $(PWD)/sed_enable.sh:usr/local/bin/sed_enable.sh \
		-files $(PWD)/sed_disable.sh:usr/local/bin/sed_disable.sh \
		-files $(PWD)/b.sh:usr/local/bin/b.sh \
		core \
		boot \
		"$(PWD)/pbainit" \
		"$(PWD)/boot2" \
		cmds/exp/partprobe \
	)
