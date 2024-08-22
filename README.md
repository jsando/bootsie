# Bootsie

Experimental utility to build an EFI boot partition in a disk image using go-diskfs (instead of mtools and/or loopback mounts).

Assuming you have your linux kernel, initrd, and whatever other files you need you can point this program
at the folder with them and it will install them into a disk image with an EFI partition.

