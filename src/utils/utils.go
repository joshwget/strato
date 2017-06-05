package utils

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/joshwget/strato/src/config"
)

// TODO: not exactly thread safe...
var Size float64

func ExtractTar(reader io.Reader, target string, whitelist, blacklist []*regexp.Regexp) error {
	return TarForEach(reader, whitelist, blacklist, writeFile(target))
}

func ExtractGzipTar(reader io.Reader, target string, whitelist, blacklist []*regexp.Regexp) error {
	return GzipTarForEach(reader, whitelist, blacklist, writeFile(target))
}

func writeFile(target string) func(io.Reader, *tar.Header) error {
	return func(tarReader io.Reader, header *tar.Header) error {
		filename := path.Join(target, header.Name)
		fmt.Println(filename)
		Size += float64(header.FileInfo().Size())

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(filename, os.FileMode(header.Mode)); err != nil {
				return err
			}
		case tar.TypeReg:
			if _, err := os.Stat(filename); err == nil {
				if err := os.Remove(filename); err != nil {
					return err
				}
			}
			writer, err := os.Create(filename)
			if err != nil {
				return err
			}
			io.Copy(writer, tarReader)
			if err = os.Chmod(filename, header.FileInfo().Mode()); err != nil {
				return err
			}
			writer.Close()
		case tar.TypeLink:
			if _, err := os.Stat(filename); err == nil {
				if err := os.Remove(filename); err != nil {
					return err
				}
			}
			if err := os.Link(header.Linkname, filename); err != nil {
				return err
			}
		case tar.TypeSymlink:
			if _, err := os.Stat(filename); err == nil {
				if err := os.Remove(filename); err != nil {
					return err
				}
			}
			if err := os.Symlink(header.Linkname, filename); err != nil {
				return err
			}
		default:
			return fmt.Errorf("Failed to untar %s (%c)", filename, header.Typeflag)
		}
		return nil
	}
}

func GzipTarForEach(reader io.Reader, whitelist, blacklist []*regexp.Regexp, f func(io.Reader, *tar.Header) error) error {
	gzipReader, err := gzip.NewReader(reader)
	if err != nil {
		return err
	}
	return TarForEach(gzipReader, whitelist, blacklist, f)
}

func TarForEach(reader io.Reader, whitelist, blacklist []*regexp.Regexp, f func(io.Reader, *tar.Header) error) error {
	tarReader := tar.NewReader(reader)
	for {
		header, err := tarReader.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		filename := header.Name
		if filename == config.Filename {
			continue
		}
		if len(whitelist) > 0 {
			passes := false
			for _, r := range whitelist {
				if r.MatchString(filename) {
					passes = true
				}
			}
			if !passes {
				continue
			}
		}
		passes := true
		for _, r := range blacklist {
			if r.MatchString(filename) {
				passes = false
			}
		}
		if !passes {
			continue
		}
		// Temporarily ignored conditions
		if strings.HasSuffix(filename, ".a") {
			continue
		}
		if strings.HasPrefix(filename, "tmp/") {
			continue
		}
		if strings.HasPrefix(filename, "usr/src/") {
			continue
		}

		if err := f(tarReader, header); err != nil {
			return err
		}
	}

	return nil
}

// TODO: write my own
func Tar(src string, writers ...io.Writer) error {

	// ensure the src actually exists before trying to tar it
	if _, err := os.Stat(src); err != nil {
		return fmt.Errorf("Unable to tar files - %v", err.Error())
	}

	mw := io.MultiWriter(writers...)

	gzw := gzip.NewWriter(mw)
	defer gzw.Close()

	tw := tar.NewWriter(gzw)
	defer tw.Close()

	// walk path
	return filepath.Walk(src, func(file string, fi os.FileInfo, err error) error {

		// return on any error
		if err != nil {
			return err
		}

		// create a new dir/file header
		header, err := tar.FileInfoHeader(fi, fi.Name())
		if err != nil {
			return err
		}

		// update the name to correctly reflect the desired destination when untaring
		header.Name = strings.TrimPrefix(strings.Replace(file, src, "", -1), string(filepath.Separator))

		// write the header
		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		// return on directories since there will be no content to tar
		if fi.Mode().IsDir() {
			return nil
		}

		// open files for taring
		f, err := os.Open(file)
		defer f.Close()
		if err != nil {
			return err
		}

		// copy file data into tar writer
		if _, err := io.Copy(tw, f); err != nil {
			return err
		}

		return nil
	})
}
