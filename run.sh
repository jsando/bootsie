#!/usr/bin/env bash
set -Eeuo pipefail

qemu-system-x86_64 -m 512m \
    -drive if=pflash,format=raw,file=/opt/homebrew/Cellar/qemu/8.1.3/share/qemu/edk2-x86_64-code.fd \
    -drive format=raw,file=disk.img,if=ide,media=disk -serial stdio
