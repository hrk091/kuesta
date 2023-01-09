package install

import (
	"encoding/yaml"
	"list"
	"strings"
	"tool/file"
	"tool/exec"
	"tool/cli"
)

command: install: {

	$usage: "cue cmd install"
	$short: "Install kuesta to Kubernetes cluster with kubectl/kustomize."

	configRepo: cli.Ask & {
		prompt:   "Github repository for config:"
		response: string
	}

	statusRepo: cli.Ask & {
		$dep:     configRepo
		prompt:   "Github repository for status:"
		response: string
	}

	usePrivateRepo: cli.Ask & {
		$dep:     statusRepo
		prompt:   "Are these repositories private? (yes|no):"
		response: bool | *false
	}

	gitToken: {
		if usePrivateRepo.response {
			cli.Ask & {
				prompt:   "Github private access token:"
				response: string
			}
		}
	}

	wantEmulator: {
		$dep: gitToken.$done
		cli.Ask & {
			prompt:   "Do you need sample driver and emulator for trial?: (yes|no)"
			response: bool
		}
	}

	printInputs: cli.Print & {
		$dep: wantEmulator.$done
		text: strings.Join([
			"",
			"---",
			"Github Config Repository: \(configRepo.response)",
			"Github Status Repository: \(statusRepo.response)",
			"Use Private Repo: \(usePrivateRepo.response)",
			if usePrivateRepo.response {
				"Github Access Token: ***"
			},
			"",
			"Deploy sample driver and emulator: \(wantEmulator.response)",
			"---",
			"",
		], "\n")
	}

	confirm: cli.Ask & {
		$dep:     printInputs.$done
		prompt:   "Continue? (yes|no)"
		response: bool | *false
	}

	printConfirmResult: cli.Print & {
		$dep: confirm.$done
		if confirm.response {
			text: "\nApplying kustomize manifests...\n"
		}
		if !confirm.response {
			text: "\nCancelled.\n"
		}
	}

	apply: {
		if confirm.response {
			vendor: deployVendor & {
				$dep: confirm.$done
			}

			wait: exec.Run & {
				$dep: vendor.$done
				cmd: ["sleep", "3"]
			}

			kuesta: deployKuesta & {
				$dep: wait.$done
				var: {
					"configRepo":     configRepo.response
					"statusRepo":     statusRepo.response
					"usePrivateRepo": usePrivateRepo.response
					if usePrivateRepo.response {
						"gitToken": gitToken.response
					}
				}
			}
			provisioner: deployProvisioner & {
				$dep: kuesta.$done
			}
			if wantEmulator.response {
				deviceOperator: deployDeviceOperator & {
					$dep: provisioner.$done
					var: {
						"statusRepo": statusRepo.response
					}
				}
				gettingStartedResources: deployGettingStartedResources & {
					$dep: deviceOperator.$done
					var: {
						"configRepo":     configRepo.response
						"usePrivateRepo": usePrivateRepo.response
						if usePrivateRepo.response {
							"gitToken": gitToken.response
						}
					}
				}
			}
		}
	}
}

deployVendor: {
	_dep="$dep": _

	start: cli.Print & {
		$dep: _dep
		text: """
			\n\n
			==============================
			Deploy vendor dependencies\n
			"""
	}

	deploy: exec.Run & {
		$dep: start.$done
		dir:  "../.."
		cmd: ["bash", "-c", """
			kubectl apply -f ./config/vendor
			kubectl apply -f ./config/privateCA
			"""]
	}

	$done: deploy.$done
}

deployKuesta: {
	_dep="$dep": _

	// inputs
	var: {
		configRepo:     string
		statusRepo:     string
		usePrivateRepo: bool
		gitToken:       string | *""
		version:        string | *"latest"
		image:          string | *"ghcr.io/nttcom-ic/kuesta/kuesta"
	}

	// private variables
	_secretEnvFileName: ".env.secret"
	_secretKeyGitToken: "gitToken"
	_k:                 kustomizations.kuesta & {
		"var": {
			configRepo:        var.configRepo
			statusRepo:        var.statusRepo
			usePrivateRepo:    var.usePrivateRepo
			secretEnvFileName: _secretEnvFileName
			secretKeyGitToken: _secretKeyGitToken
		}
	}
	_kustomizationFile: strings.Join([_k.path, "kustomization.yaml"], "/")
	_patchFile:         strings.Join([_k.path, "patch.yaml"], "/")
	_secretEnvFile:     strings.Join([_k.path, _secretEnvFileName], "/")

	// tasks
	start: cli.Print & {
		$dep: _dep
		text: """
			\n\n
			==============================
			Deploy kuesta\n
			"""
	}

	mkdir: file.MkdirAll & {
		$dep: start.$done
		path: _k.path
	}

	writeKustomization: file.Create & {
		$dep:     mkdir.$done
		filename: _kustomizationFile
		contents: yaml.Marshal(_k.kustomization)
	}

	writePatch: file.Create & {
		$dep:     mkdir.$done
		filename: _patchFile
		contents: yaml.MarshalStream([ for _, v in _k.patches {v}])
	}

	writeSecret: {
		$dep: mkdir.$done
		if var.usePrivateRepo {
			file.Create & {
				$dep:     writePatch.$done
				filename: _secretEnvFile
				contents: "\(_secretKeyGitToken)=\(var.gitToken)"
			}
		}
	}

	deploy: exec.Run & {
		$dep: [writeKustomization.$done, writePatch.$done, writeSecret.$done]
		dir: _k.baseDir
		cmd: ["bash", "-c", """
			export IMG='\(var.image):\(var.version)'
			export KUSTOMIZE_ROOT='\(_k.kustomizeRoot)'
			make deploy
			"""]
	}

	$done: deploy.$done
}

deployProvisioner: {
	_dep="$dep": _

	// input
	var: {
		version: string | *"latest"
		image:   string | *"ghcr.io/nttcom-ic/kuesta/provisioner"
	}

	// private variables
	_k: kustomizations.provisioner

	// tasks
	start: cli.Print & {
		$dep: _dep
		text: """
			\n\n
			==============================
			Deploy kuesta-provisioner\n
			"""
	}

	installCRD: exec.Run & {
		$dep: start.$done
		dir:  _k.baseDir
		cmd: ["bash", "-c", "make install"]
	}

	deploy: exec.Run & {
		$dep: installCRD.$done
		dir:  _k.baseDir
		cmd: ["bash", "-c", """
			export IMG='\(var.image):\(var.version)'
			export KUSTOMIZE_ROOT='\(_k.kustomizeRoot)'
			make deploy
			"""]
	}

	$done: deploy.$done
}

deployDeviceOperator: {
	_dep="$dep": _

	// inputs
	var: {
		statusRepo:      string
		version:         string | *"latest"
		image:           string | *"ghcr.io/nttcom-ic/kuesta/device-operator"
		subscriberImage: string | *"ghcr.io/nttcom-ic/kuesta/device-subscriber"
	}

	// private variables
	_k: kustomizations.deviceoperator & {
		"var": {
			statusRepo:      var.statusRepo
			version:         var.version
			subscriberImage: var.subscriberImage
		}
	}
	_kustomizationFile: strings.Join([_k.path, "kustomization.yaml"], "/")
	_patchFile:         strings.Join([_k.path, "patch.yaml"], "/")

	// tasks
	start: cli.Print & {
		$dep: _dep
		text: """
			\n\n
			==============================
			Deploy device-operator\n
			"""
	}

	mkdir: file.MkdirAll & {
		$dep: start.$done
		path: _k.path
	}

	writeKustomization: file.Create & {
		$dep:     mkdir.$done
		filename: _kustomizationFile
		contents: yaml.Marshal(_k.kustomization)
	}

	writePatch: file.Create & {
		$dep:     mkdir.$done
		filename: _patchFile
		contents: yaml.MarshalStream([ for _, v in _k.patches {v}])
	}

	installCRD: exec.Run & {
		$dep: [writeKustomization.$done, writePatch.$done]
		dir: _k.baseDir
		cmd: ["bash", "-c", "make install"]
	}

	deploy: exec.Run & {
		$dep: installCRD.$done
		dir:  _k.baseDir
		cmd: ["bash", "-c", """
			export IMG='\(var.image):\(var.version)'
			export KUSTOMIZE_ROOT='\(_k.kustomizeRoot)'
			make deploy
			"""]
	}

	$done: deploy.$done
}

deployGettingStartedResources: {
	_dep="$dep": _

	// inputs
	var: {
		namespace:      string | *"kuesta-getting-started"
		configRepo:     string
		usePrivateRepo: bool
		gitToken:       string | *""
		gnmiFakeImage:  string | *"ghcr.io/nttcom-ic/kuesta/gnmi-fake:latest"
	}

	// private variables
	let _manifestFile = "getting-started.yaml"
	let _resources = [
		resources.namespace & {
			"var": namespace: var.namespace
		},
		resources.gitRepository & {
			"var": {
				namespace:      var.namespace
				configRepo:     var.configRepo
				usePrivateRepo: var.usePrivateRepo
				if var.usePrivateRepo {
					gitRepoSecretRef: "TBD"
				}
			}
		},
		resources.deviceOcDemo & {
			"var": {
				name:       "oc01"
				namespace:  var.namespace
				configRepo: var.configRepo
			}
		},
		resources.deviceOcDemo & {
			"var": {
				name:       "oc02"
				namespace:  var.namespace
				configRepo: var.configRepo
			}
		},
		resources.gnmiFake & {
			"var": {
				name:      "oc01"
				namespace: var.namespace
				image:     var.gnmiFakeImage
			}
		},
		resources.gnmiFake & {
			"var": {
				name:      "oc02"
				namespace: var.namespace
				image:     var.gnmiFakeImage
			}
		},
	]

	// tasks
	start: cli.Print & {
		$dep: _dep
		text: """
			\n\n
			==============================
			Deploy getting-started resources\n
			"""
	}

	writeManifest: file.Create & {
		$dep:     start.$done
		filename: _manifestFile
		contents: yaml.MarshalStream(list.Concat([ for _, v in _resources {v.out}]))
	}

	deploy: exec.Run & {
		$dep: writeManifest.$done
		cmd: ["bash", "-c", """
			kubectl apply -f \(_manifestFile)
			rm -f \(_manifestFile)
			"""]
	}

	$done: deploy.$done
}
