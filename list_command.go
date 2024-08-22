package main

import (
	"flag"
	"fmt"
	"github.com/diskfs/go-diskfs"
	"github.com/diskfs/go-diskfs/filesystem"
	gzip "github.com/klauspost/pgzip"
	"io"
	"os"
	"path/filepath"
)

type ListCommand struct {
	fs        *flag.FlagSet
	imageFile string
	longForm  bool
}

func NewListCommand() *ListCommand {
	c := &ListCommand{
		fs: flag.NewFlagSet("list", flag.ExitOnError),
	}
	c.fs.BoolVar(&c.longForm, "long", false, "List file attributes similar to ls -l")
	c.fs.Usage = func() {
		const instruction string = `
List EFI partition contents, recursively

`
		fmt.Fprintf(os.Stderr, instruction)
		fmt.Fprintf(os.Stderr, "Usage: %s [options] <path>\n", os.Args[0])
		c.fs.PrintDefaults()
	}
	return c
}

func (c *ListCommand) Name() string {
	return c.fs.Name()
}

func (c *ListCommand) Run(args []string) error {
	err := c.fs.Parse(args)
	if err != nil {
		return err
	}
	if len(c.fs.Args()) != 1 {
		return fmt.Errorf("expected path as last argument")
	}
	c.imageFile = c.fs.Arg(0)
	// if filename ends with ".gz", gunzip to a tempfile
	if filepath.Ext(c.imageFile) == ".gz" {
		tempFile, err := gunzipToTempFile(c.imageFile)
		if err != nil {
			return fmt.Errorf("could not gunzip file: %s", err)
		}
		c.imageFile = tempFile
		defer os.Remove(tempFile)
	}
	disk, err := diskfs.Open(c.imageFile)
	fs, err := disk.GetFilesystem(1)
	if err != nil {
		return fmt.Errorf("could not open filesystem: %s", err)
	}
	return c.listDir(fs, "/")
}

func gunzipToTempFile(filename string) (tempFilename string, err error) {
	tempFile, err := os.CreateTemp(os.TempDir(), "disk.img.")
	if err != nil {
		return "", err
	}
	defer tempFile.Close()
	fmt.Fprintf(os.Stderr, "uncompressing %s to temp file %s ...\n", filename, tempFile.Name())
	reader, err := os.Open(filename)
	if err != nil {
		return tempFile.Name(), err
	}
	gzipReader, err := gzip.NewReader(reader)
	defer gzipReader.Close()
	if err != nil {
		return tempFile.Name(), err
	}
	_, err = io.Copy(tempFile, gzipReader)
	return tempFile.Name(), err
}

func (c *ListCommand) listDir(fs filesystem.FileSystem, path string) error {
	files, err := fs.ReadDir(path)
	if err != nil {
		return err
	}
	for _, file := range files {
		if file.Name() == "." || file.Name() == ".." {
			continue
		}
		absPath := filepath.Join(path, file.Name())
		fmt.Printf("%s\n", absPath)
		if file.IsDir() {
			err = c.listDir(fs, absPath)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
