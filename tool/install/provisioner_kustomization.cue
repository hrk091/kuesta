package install

import (
	"strings"
)

kustomizations: {
	provisioner: {
		// path
		baseDir:       "../../provisioner"
		configDir:     "config"
		kustomizeRoot: "default"
		path:          strings.Join([baseDir, configDir, kustomizeRoot], "/")
	}
}
