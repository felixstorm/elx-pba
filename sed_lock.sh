#!/bin/sh

sedutil-cli --setLockingRange 0 lk $args[0] /dev/nvme0
sedutil-cli --setMbrDone off $args[0] /dev/nvme0
