package index

import (
	"io/ioutil"
	"path"
	"strings"

	"github.com/joshwget/strato/src/config"
	"gopkg.in/yaml.v2"
)

func Generate(dir string) ([]byte, error) {
	packageMap := map[string]config.Package{}

	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		b, err := ioutil.ReadFile(path.Join(dir, file.Name(), config.Filename))
		if err != nil {
			return nil, err
		}

		var pkg config.Package
		if err := yaml.Unmarshal(b, &pkg); err != nil {
			return nil, err
		}

		packageName := file.Name()
		if strings.Contains(packageName, ".") {
			packageName = strings.SplitN(packageName, ".", 2)[1]
		}

		packageMap[packageName] = config.Package{
			Dependencies: pkg.Dependencies,
		}

		for subpackageName, subpackage := range pkg.Subpackages {
			packageMap[subpackageName] = config.Package{
				Dependencies: subpackage.Dependencies,
			}
		}
	}

	return yaml.Marshal(packageMap)
}
