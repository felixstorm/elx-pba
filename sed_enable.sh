#!/bin/sh

sedutil-cli --enableLockingRange 0 $args[0] /dev/nvme0
sedutil-cli --setMbrEnable on $args[0] /dev/nvme0
