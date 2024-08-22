package main

import (
	"flag"
	"fmt"
	"github.com/diskfs/go-diskfs"
	"github.com/diskfs/go-diskfs/filesystem"
	"io"
	"os"
	"path/filepath"
)

type CopyCommand struct {
	fs        *flag.FlagSet
	imageFile string
	destDir   string
}

func NewCopyCommand() *CopyCommand {
	c := &CopyCommand{
		fs: flag.NewFlagSet("cp", flag.ExitOnError),
	}
	c.fs.Usage = func() {
		const instruction string = `
Copy source to dest, recursively
Currently source must be disk image, and dest must be a folder.

`
		fmt.Fprintf(os.Stderr, instruction)
		fmt.Fprintf(os.Stderr, "Usage: %s [options] <source-path> <dest-path>\n", os.Args[0])
		c.fs.PrintDefaults()
	}
	return c
}

func (c *CopyCommand) Name() string {
	return c.fs.Name()
}

func (c *CopyCommand) Run(args []string) error {
	err := c.fs.Parse(args)
	if err != nil {
		return err
	}
	if len(c.fs.Args()) != 2 {
		return fmt.Errorf("expected <source> and <dest> arguments")
	}
	c.imageFile = c.fs.Arg(0)
	c.destDir = c.fs.Arg(1)

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

	// Ensure destDir exists
	err = os.MkdirAll(c.destDir, 0755)
	if err != nil {
		return fmt.Errorf("could not create destination directory: %s", err)
	}
	return c.copyFiles(fs, "/")
}

func (c *CopyCommand) copyFiles(fs filesystem.FileSystem, sourceDir string) error {
	files, err := fs.ReadDir(sourceDir)
	if err != nil {
		return err
	}
	for _, file := range files {
		if file.Name() == "." || file.Name() == ".." {
			continue
		}
		absPath := filepath.Join(sourceDir, file.Name())
		targetPath := filepath.Join(c.destDir, absPath)
		err = os.MkdirAll(filepath.Dir(targetPath), 0755)
		if err != nil {
			return fmt.Errorf("could not create directory for target path: %s", err)
		}
		if file.IsDir() {
			err = c.copyFiles(fs, absPath)
		} else {
			fmt.Printf("Writing %s ... \n", targetPath)
			err = c.copyFile(fs, absPath, targetPath)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *CopyCommand) copyFile(fs filesystem.FileSystem, absPath string, targetPath string) error {
	sourceFile, err := fs.OpenFile(absPath, os.O_RDONLY)
	if err != nil {
		return fmt.Errorf("could not open source file: %s", err)
	}
	defer sourceFile.Close()

	destFile, err := os.Create(targetPath)
	if err != nil {
		return fmt.Errorf("could not create destination file: %s", err)
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return fmt.Errorf("could not copy file: %s", err)
	}
	return nil
}
