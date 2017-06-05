package index

import (
	"io/ioutil"
	"path"

	"github.com/joshwget/strato/src/pkg/index"
	"github.com/urfave/cli"
)

func Action(c *cli.Context) error {
	inDir := c.Args()[0]
	outDir := c.Args()[1]

	indexBytes, err := index.GenerateIndex(inDir)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(path.Join(outDir, "index.yml"), indexBytes, 0644)
}
