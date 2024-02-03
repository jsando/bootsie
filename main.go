package main

import (
	"flag"
	"fmt"
	"github.com/diskfs/go-diskfs"
	"github.com/diskfs/go-diskfs/disk"
	"github.com/diskfs/go-diskfs/filesystem"
	"github.com/diskfs/go-diskfs/partition/mbr"
	gzip "github.com/klauspost/pgzip"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	outputPath := flag.String("output", "disk.img", "output path")
	partitionMB := flag.Int("size", 1024, "partition size in megabytes")
	label := flag.String("label", "boot", "volume label")
	force := flag.Bool("force", false, "force overwrite if output path exists")
	doGzip := flag.Bool("gzip", false, "compress output file with gzip")
	flag.Parse()

	// Delete the output file if it exists already and -force was specified
	if _, err := os.Stat(*outputPath); err == nil {
		if !*force {
			fmt.Fprintf(os.Stderr, "Output path '%s' exists, remove it or use --force to overwrite\n", *outputPath)
			os.Exit(1)
		}
		os.Remove(*outputPath)
	}

	// Package everything into an EFI partition
	err := createDiskImage(*partitionMB, *outputPath, *label, flag.Args())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating disk image: %s\n", err.Error())
		os.Exit(1)
	}

	// Optionally compress the output
	if *doGzip {
		fmt.Fprintf(os.Stderr, "Compressing %s ... \n", *outputPath)
		err = compressOutput(outputPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error compressing output: %s\n", err.Error())
			os.Exit(1)
		}
	}
}

func compressOutput(outputPath *string) error {
	gzipFilename := fmt.Sprintf("%s.gz", *outputPath)
	gzipFile, err := os.Create(gzipFilename)
	if err != nil {
		panic(err)
	}
	reader, err := os.Open(*outputPath)
	defer reader.Close()
	w := gzip.NewWriter(gzipFile)
	//w.SetConcurrency(100000, 10)
	_, err = io.Copy(w, reader)
	err2 := w.Close()
	if err != nil {
		return err
	}
	return err2
}

const MB = 1024 * 1024
const FATBlockSize = 512
const FATPartitionSTart = 2048

func createDiskImage(partitionMB int, outputPath string, volumeLabel string, includes []string) error {
	espSize := partitionMB * MB
	diskSize := espSize + 4*MB
	partitionSectors := espSize / FATBlockSize

	// create raw disk image file
	myDisk, err := diskfs.Create(outputPath, int64(diskSize), diskfs.Raw, diskfs.SectorSizeDefault)
	if err != nil {
		return err
	}

	// create a partition table
	table := &mbr.Table{
		Partitions: []*mbr.Partition{
			{Start: FATPartitionSTart, Size: uint32(partitionSectors), Type: mbr.EFISystem, Bootable: true},
		},
	}
	err = myDisk.Partition(table)
	if err != nil {
		return err
	}

	spec := disk.FilesystemSpec{Partition: 1, FSType: filesystem.TypeFat32, VolumeLabel: volumeLabel}
	fs, err := myDisk.CreateFilesystem(spec)
	if err != nil {
		return err
	}

	for _, include := range includes {
		paths, err := filepath.Glob(include)
		if err != nil {
			return err
		}
		for _, path := range paths {
			// If its a folder, copy it recursively including the folder name in the destination
			finfo, err := os.Stat(path)
			if err != nil {
				return fmt.Errorf("error accessing file '%s': %s\n", path, err.Error())
			}
			if finfo.IsDir() {
				prefix := filepath.Dir(path)
				err = copyDir(prefix, path, fs)
				if err != nil {
					return err
				}
			} else {
				// If its a file, copy it straight over
				err = copyFile(path, "/"+filepath.Base(path), fs)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// /a/b/c/ --> 			c/
// /a/b/c/d/ --> 		c/d/
// /a/b/c/f.txt -> 		c/f.txt
// this is re-rooting folder c to / from /a/b ... therefore the trick is to
// pass in the prefix to subtract
func copyDir(prefix string, path string, fs filesystem.FileSystem) error {
	files, err := os.ReadDir(path)
	if err != nil {
		return err
	}

	if !strings.HasPrefix(path, prefix) {
		return fmt.Errorf("path '%s' is not rooted in %s", path, prefix)
	}
	targetPath := path[len(prefix):]
	fs.Mkdir(targetPath)
	for _, file := range files {
		if file.IsDir() {
			copyDir(prefix, filepath.Join(path, file.Name()), fs)
		} else {
			copyFile(filepath.Join(path, file.Name()), filepath.Join(targetPath, file.Name()), fs)
		}
	}
	return nil
}

func copyFile(src string, dst string, fs filesystem.FileSystem) error {
	file, err := os.Open(src)
	fmt.Printf("Copying %s -> %s \n", src, dst)
	rw, err := fs.OpenFile(dst, os.O_CREATE|os.O_RDWR)
	if err != nil {
		return fmt.Errorf("error writing output file '%s': %s\n", dst, err.Error())
	}
	// Copy file but with a much larger buffer than 32kb default, the rootfs is > 100mb, this makes a huge
	// improvement (for a 150Mb rootfs it was 30s total vs 1s with the larger buffer).
	buf := make([]byte, 1048576)
	_, err = io.CopyBuffer(rw, file, buf)
	if err != nil {
		return err
	}
	err = rw.Close()
	if err != nil {
		return err
	}
	file.Close()
	return nil
}
