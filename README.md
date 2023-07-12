# elx-pba

Pre-boot authentication image for TCG Storage devices

Fork of the excellent work of https://github.com/elastx/elx-pba with the unlock password being queried interactively from the user instead of being taken from the system UUID.  
Since my time is pretty limited, this is meant just to make my personal changes publicly available and I will most likely not be able to react on issues or accept pull requests. But feel free to fork again and enhance it yourself.

### Important
This repository has now been customized to my personal situation, i.e. it will only try to load `grub.cfg` from `/dev/nvme0n1p2` and not from anywhere else. This is required as mounting of ext4 volumes will replay its journal even when mounted read-only if the filesystem is dirty and hence will mess up resuming from hibernation.  
Therefore I decided to move my `/boot` folder to a separate partition formatted as ext2 (which does not have a journal in contrast to ext4) and have u-root mount this to search for `grub.cfg`. Now resuming from hibernation does work well also with `kexec` and does not require rebooting after unlocking anymore.  
To adjust this to your situation change the partition name provided as a parameter to `boot2` in [pbainit/main.go](pbainit/main.go) (or completely remove it including `-name`).

### Update 2023-07-12
As I encountered a drive that could not be unlocked using `github.com/bluecmd/go-tcg-storage` (error: "admin session creation failed: invalid argument") I switched to calling `sedutil-cli` instead for unlocking from `pbainit` for now. I might dive deeper into this issue at some point in the future but since everything works well again with this change, this is not on my to-do-list with high priority.

## Building

**NOTE**: Use a Go version of 1.20.

```shell
$ sudo apt install \
    gnupg2 gpgv2 flex bison build-essential libelf-dev \
    curl libssl-dev bc zstd dosfstools fdisk gdisk mtools
$ gpg2 --locate-keys torvalds@kernel.org gregkh@kernel.org autosigner@kernel.org

# Make sure sgdisk is in the PATH
$ PATH=$PATH:/sbin make
```

Alternatively, use the containerized build tools:

```shell
$ docker build \
	-t elx-pba-builder:latest \
	-f builder.dockerfile .
$ docker run \
	--rm --volume ${PWD}:/src \
	elx-pba-builder:latest
```


## Testing in a VM

```shell
$ sudo apt install qemu-system-x86
$ make qemu-x86_64
```

## Testing on a real disk

```shell
$  export OPAL_KEY=debug # keep space before command to keep it out of your (bash) history
$ sudo arch/x86_64/sedutil/sedutil-cli --loadpbaimage "${OPAL_KEY}" .build/elx-pba-x86_64.img /dev/nvme0n1
```
