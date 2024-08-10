package main

import (
	"flag"
	"fmt"
	"github.com/diskfs/go-diskfs"
	"github.com/diskfs/go-diskfs/disk"
	"github.com/diskfs/go-diskfs/filesystem"
	"github.com/diskfs/go-diskfs/partition/mbr"
	"github.com/dustin/go-humanize"
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
	doTruncate := flag.Bool("truncate", false, "truncate disk image before compressing")
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

	// Truncate?
	if *doTruncate {
		fmt.Printf("Truncating disk image ... ")
		trimSize, err := trimFile(*outputPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error finding trimmed file size: %s\n", err.Error())
			os.Exit(1)
		}
		err = os.Truncate(*outputPath, trimSize)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error truncating disk image: %s\n", err.Error())
			os.Exit(1)
		}
		fmt.Printf("truncated image to %s\n", humanize.Bytes(uint64(trimSize)))
	}

	// Optionally compress the output
	if *doGzip {
		fmt.Fprintf(os.Stderr, "Compressing %s ... \n", *outputPath)
		err = compressOutput(outputPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error compressing output: %s\n", err.Error())
			os.Exit(1)
		}
		err = os.Remove(*outputPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error removing %s: %s\n", *outputPath, err.Error())
		}
	}
}

func trimFile(filePath string) (int64, error) {
	const chunkSize = 1048576

	file, err := os.Open(filePath)
	if err != nil {
		return -1, err
	}
	defer file.Close()

	// Get file size
	fileInfo, err := file.Stat()
	if err != nil {
		return -1, err
	}
	fileSize := fileInfo.Size()
	trimSize := fileSize

	// Read the file in reverse, in chunks
scanLoop:
	for offset := fileSize; offset > 0; offset -= chunkSize {
		// Calculate the size of the chunk to read
		readSize := int64(chunkSize)
		if offset < chunkSize {
			readSize = offset
		}

		// Move the offset back to read the chunk
		buf := make([]byte, readSize)
		_, err := file.ReadAt(buf, offset-readSize)
		if err != nil {
			return -1, err
		}

		// Scan the chunk from the end towards the beginning
		for i := len(buf) - 1; i >= 0; i-- {
			if buf[i] != 0 {
				trimSize = offset - readSize + int64(i)
				break scanLoop
			}
		}
	}

	// If no non-zero byte is found, return -1
	return trimSize, nil
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
const BlockSize = 512
const PartitionStart = 2048

func createDiskImage(partitionMB int, outputPath string, volumeLabel string, includes []string) error {
	espSize := partitionMB * MB
	diskSize := espSize + 4*MB
	partitionSectors := espSize / BlockSize
	//partitionEnd := partitionSectors - PartitionStart + 1

	// create raw disk image file
	myDisk, err := diskfs.Create(outputPath, int64(diskSize), diskfs.Raw, diskfs.SectorSizeDefault)
	if err != nil {
		return err
	}

	// create a partition table
	table := &mbr.Table{
		Partitions: []*mbr.Partition{
			{
				Start:    PartitionStart,
				Size:     uint32(partitionSectors),
				Type:     mbr.EFISystem,
				Bootable: true,
			},
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
	err = myDisk.Close()
	return err
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
	if err != nil {
		return fmt.Errorf("error opening file '%s': %s\n", src, err.Error())
	}
	defer file.Close()
	rw, err := fs.OpenFile(dst, os.O_CREATE|os.O_RDWR)
	if err != nil {
		return fmt.Errorf("error writing output file '%s': %s\n", dst, err.Error())
	}
	// Make the buffer the same size as the file so that the fat32 file can see how many clusters
	// to allocate, otherwise it will re-allocate the cluster chain size/len(buf) times.
	// See https://github.com/diskfs/go-diskfs/issues/130
	// For very large files (ie 512M) it takes several minutes, versus loading the entire file into ram takes 1s or less.
	// This is temporary until the above issue can be addressed better ie FileSystem.Truncate(name,size).
	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("error getting file info for '%s': %s\n", dst, err.Error())
	}
	fmt.Printf("Copying %s -> %s (%d bytes)\n", src, dst, info.Size())
	buf, err := io.ReadAll(file)
	n, err := rw.Write(buf)
	if err != nil {
		return err
	}
	if n != int(info.Size()) {
		return fmt.Errorf("error writing output file '%s': %d bytes written, s/b %d\n", dst, n, info.Size())
	}
	err = rw.Close()
	if err != nil {
		return err
	}
	return nil
}
