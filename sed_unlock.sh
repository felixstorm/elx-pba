#!/bin/sh

sedutil-cli --setLockingRange 0 rw $args[0] /dev/nvme0
sedutil-cli --setMbrDone on $args[0] /dev/nvme0

partprobe /dev/nvme0n1
