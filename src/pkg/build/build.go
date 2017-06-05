package build

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"

	yaml "gopkg.in/yaml.v2"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/joshwget/strato/src/config"
	"github.com/joshwget/strato/src/utils"
	"github.com/urfave/cli"
)

const (
	imageName = "package"
)

type info struct {
	Layers []string `json:"Layers"`
}

func Action(c *cli.Context) error {
	dockerfile := c.String("f")
	dir := c.String("d")

	_ = dockerfile
	_ = dir

	cli, err := client.NewEnvClient()
	if err != nil {
		return err
	}

	buildContext := bytes.Buffer{}

	if err = utils.Tar(".", &buildContext); err != nil {
		return err
	}

	buildContextReader := bytes.NewReader(buildContext.Bytes())

	imageName := "strap"

	if _, err = cli.ImageBuild(context.Background(), buildContextReader, types.ImageBuildOptions{
		Tags: []string{
			imageName,
		},
	}); err != nil {
		return err
	}

	inDir := "./"
	outDir := "./"
	configPath := path.Join(inDir, "strato.yml")

	/*packageName := path.Base(inDir)
	if strings.Contains(packageName, ".") {
		packageName = strings.SplitN(packageName, ".", 2)[1]
	}*/

	packageName := "josh"

	b, err := ioutil.ReadFile(configPath)
	if err != nil {
		return err
	}

	var pkg config.Package
	if err := yaml.Unmarshal(b, &pkg); err != nil {
		return err
	}

	reader, err := cli.ImageSave(context.Background(), []string{imageName})
	if err != nil {
		return err
	}

	buf := new(bytes.Buffer)
	if err := utils.TarForEach(reader, nil, nil, func(tarReader io.Reader, header *tar.Header) error {
		if header.Name == "manifest.json" {
			io.Copy(buf, tarReader)
		}
		return nil
	}); err != nil {
		return err
	}

	var infos []info
	if err := json.Unmarshal(buf.Bytes(), &infos); err != nil {
		return err
	}

	layers := infos[0].Layers
	layer := layers[len(layers)-1]

	reader.Close()

	reader, err = cli.ImageSave(context.Background(), []string{imageName})
	if err != nil {
		return err
	}

	buf = new(bytes.Buffer)
	if err := utils.TarForEach(reader, nil, nil, func(tarReader io.Reader, header *tar.Header) error {
		if header.Name == layer {
			io.Copy(buf, tarReader)
		}
		return nil
	}); err != nil {
		return err
	}

	b = buf.Bytes()
	if err = generatePackage(b, outDir, packageName, &pkg); err != nil {
		return err
	}
	for subpackageName := range pkg.Subpackages {
		if err = generatePackage(b, outDir, subpackageName, &pkg); err != nil {
			return err
		}
	}

	return nil
}

func generatePackage(b []byte, outDir, name string, pkg *config.Package) error {
	// TODO: make the default package code more obvious
	whitelist, blacklist, err := config.GenerateWhiteAndBlackLists(pkg, name)
	if err != nil {
		return err
	}

	f, err := os.Create(path.Join(outDir, name) + ".tar.gz")
	if err != nil {
		return err
	}
	gzipWriter := gzip.NewWriter(f)
	packageWriter := tar.NewWriter(gzipWriter)

	layerReader := bytes.NewReader(b)
	if err := utils.TarForEach(layerReader, whitelist, blacklist, func(tarReader io.Reader, header *tar.Header) error {
		fmt.Printf("%s | %s\n", name, header.Name)
		packageWriter.WriteHeader(header)
		buf := new(bytes.Buffer)
		io.Copy(buf, tarReader)
		packageWriter.Write(buf.Bytes())
		return nil
	}); err != nil {
		return err
	}

	packageWriter.Close()
	gzipWriter.Close()
	f.Close()

	return nil
}
