package install

import (
	"encoding/yaml"
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
			kuesta: deployKuesta & {
				var: {
					"configRepo":     configRepo.response
					"statusRepo":     statusRepo.response
					"usePrivateRepo": usePrivateRepo.response
					if usePrivateRepo.response {
						"gitToken": gitToken.response
					}
				}
			}
			provisioner: deployProvisioner
			if wantEmulator.response {
				deviceOperator: deployDeviceOperator & {
					var: {
						"statusRepo": statusRepo.response
					}
				}
			}
		}
	}
}

deployKuesta: {
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
	mkdir: file.MkdirAll & {
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

	writeSecret: file.Create & {
		$dep:     mkdir.$done
		filename: _secretEnvFile
		contents: "\(_secretKeyGitToken)=\(var.gitToken)"
	}

	deploy: exec.Run & {
		$dep: writePatch.$done
		dir:  _k.baseDir
		cmd: ["bash", "-c", """
			export IMG='\(var.image):\(var.version)'
			export KUSTOMIZE_ROOT='\(_k.kustomizeRoot)'
			make deploy-preview
			"""]
	}
}

deployProvisioner: {
	// input
	var: {
		version: string | *"latest"
		image:   string | *"ghcr.io/nttcom-ic/kuesta/provisioner"
	}

	// private variables
	_k: kustomizations.provisioner

	// tasks
	deploy: exec.Run & {
		dir: _k.baseDir
		cmd: ["bash", "-c", """
			export IMG='\(var.image):\(var.version)'
			export KUSTOMIZE_ROOT='\(_k.kustomizeRoot)'
			make deploy-preview
			"""]
	}
}

deployDeviceOperator: {
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
	mkdir: file.MkdirAll & {
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

	deploy: exec.Run & {
		$dep: writePatch.$done
		dir:  _k.baseDir
		cmd: ["bash", "-c", """
			export IMG='\(var.image):\(var.version)'
			export KUSTOMIZE_ROOT='\(_k.kustomizeRoot)'
			make deploy-preview
			"""]
	}
}
