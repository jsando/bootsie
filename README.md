# fatimg

Experimental utility to build an fat32 boot (EFI) partition in a disk image using the excellent go-diskfs library, 
instead of mtools or loopback mounts.

Assuming you have your linux kernel, initrd, and whatever other files you need you can point this program
at the folder with them and it will install them into a disk image with an EFI partition.
 
fatimg can create, extract, and list files on the first partition (which must be FAT32) of a disk image.

The disk image file can optionally be gzipped, in which case fatimg will automatically gunzip it to a tmp file.

WARNING: there will probably be breaking changes as I intend to extend it to support legacy boot images with MBR,
as well as GPT (so additional cli flags to specify what is now assumed defaults).

