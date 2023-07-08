ifeq ($(ARCH),x86_64)
KERNEL_IMAGE := .build/linux-$(LINUX_VERSION)/arch/x86_64/boot/bzImage
endif

.build/linux-$(LINUX_VERSION).tar.xz:
	(mkdir -p .build; cd .build; "$(PWD)/get-verified-tarball.sh" "$(LINUX_VERSION)" || (rm -f "$@"; exit 1) )

.build/linux-$(LINUX_VERSION)/.dir: .build/linux-$(LINUX_VERSION).tar.xz
	tar -xf .build/linux-$(LINUX_VERSION).tar.xz -C .build
	touch .build/linux-$(LINUX_VERSION)/.dir

.build/linux-$(LINUX_VERSION)/.config: .build/linux-$(LINUX_VERSION)/.dir arch/$(ARCH)/linux.config
	cp -v "$(PWD)/arch/$(ARCH)/linux.config" "$@"
	(cd .build/linux-$(LINUX_VERSION); make ARCH="$(ARCH)" olddefconfig)

.PHONY: linux
linux:
	make -C .build/linux-$(LINUX_VERSION) ARCH="$(ARCH)" all -j $(shell nproc)

$(KERNEL_IMAGE): .build/linux-$(LINUX_VERSION)/.config .build/rootfs-$(ARCH).cpio
	make ARCH="$(ARCH)" LINUX_VERSION="$(LINUX_VERSION)" linux
	touch "$(@)"
