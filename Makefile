ARCH ?= $(shell uname -m)
# latest as of 23-12-06
LINUX_VERSION ?= 5.10.202
UROOT_GIT_REF ?= 722eeaf
GO_TCG_STORAGE_GIT_REF ?= 8741725

LOCAL_GIT_INFO ?= $(shell git log --pretty=format:'%h (%ci, %D)' -n 1)
GOPATH ?= $(PWD)/.build/go

ifeq ($(shell uname),Linux)
ACCEL ?= kvm
else ifeq ($(shell uname),Darwin)
ACCEL ?= hvf
else
ACCEL ?= tcg
endif

.PHONY: all
all: .build/elx-pba-$(ARCH).img

.DELETE_ON_ERROR:

include kernel.mk
include u-root.mk
include rootfs.mk
include image.mk

.PHONY: qemu-x86_64
qemu-x86_64: .build/elx-rescue-x86_64.img arch/x86_64/ovmf.fd
	qemu-system-x86_64 \
		-m 1024 \
		-uuid 00000000-0000-0000-0000-000000000001 \
		-smbios type=1,serial=SYSTEM01 \
		-smbios type=2,serial=BOARD01 \
		-smbios type=3,serial=CHASSIS01 \
		-device "virtio-scsi-pci,id=scsi0" \
		-device "scsi-hd,bus=scsi0.0,drive=hd0" \
		-drive "id=hd0,if=none,format=raw,readonly=on,file=$<" \
		-drive "if=pflash,format=raw,readonly=on,file=arch/x86_64/ovmf.fd" \
		-accel "$(ACCEL)" \
		-machine "type=q35,smm=on,usb=on" \
		-no-reboot

.PHONY: clean
clean:
	rm -vf .build/rootfs-*.cpio .build/elx-*.fs .build/elx-*.img

.PHONY: clean-go
clean-go: clean
	if [ -e .build/go ]; then chmod -R +rw .build/go; fi
	rm -rf .build/go

.PHONY: clean-deep
clean-deep:
	if [ -e .build/* ]; then chmod -R +rw .build/*; fi
	rm -rf .build/*
