# Bootsie

Experimental utility to build an EFI boot partition in a disk image using go-diskfs (instead of mtools and/or loopback mounts).

Assuming you have your linux kernel, initrd, and whatever other files you need you can point this program
at the folder with them and it will install them into a disk image with an EFI partition.

To test the built image in qemu (on aarch64 mac):

```bash
qemu-system-x86_64 -m 512m \
-drive if=pflash,format=raw,file=/opt/homebrew/Cellar/qemu/8.1.3/share/qemu/edk2-x86_64-code.fd \
-drive format=raw,file=disk.img,if=ide,media=disk -serial stdio
```

TODO:
- Add error if same file or folder names but different case (/efi vs /EFI)
- Fix cli to have short and long form and proper help
- Don't require path to have trailing slash
- Add option to specify GPT or MBR
  - MBR allows to trim to only data size written, GPT appears to write a backup table at the end of the device?
- [x] If doing -gzip delete the disk.img once the gzip is created
- [x] Option to truncate disk.img to just data space
