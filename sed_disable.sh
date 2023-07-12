#!/bin/sh

sedutil-cli --disableLockingRange 0 $args[0] /dev/nvme0
sedutil-cli --setMbrEnable off $args[0] /dev/nvme0

partprobe /dev/nvme0n1
