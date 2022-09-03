IMG ?= nwctl:latest
KUSTOMIZE_ROOT ?= default

.PHONY: docker-build
docker-build: test
	docker build -f build/Dockerfile.nwctl -t ${IMG} .

.PHONY: docker-push
docker-push:
	docker push ${IMG}

.PHONY: manifests
manifests: kustomize
	cd config/bases/nwctl && $(KUSTOMIZE) edit set image nwctl=${IMG}
	kubectl kustomize config/${KUSTOMIZE_ROOT}

.PHONY: install
install: kustomize
	cd config/bases/nwctl && $(KUSTOMIZE) edit set image nwctl=${IMG}
	kubectl apply -k config/${KUSTOMIZE_ROOT}

.PHONY: uninstall
uninstall: kustomize
	cd config/bases/nwctl && $(KUSTOMIZE) edit set image nwctl=${IMG}
	kubectl delete -k config/${KUSTOMIZE_ROOT}

##@ Build Dependencies

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Binaries
KUSTOMIZE ?= $(LOCALBIN)/kustomize
KUSTOMIZE_INSTALL_SCRIPT ?= "https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh"

## Tool Versions
KUSTOMIZE_VERSION ?= v3.8.7

.PHONY: kustomize
kustomize: $(KUSTOMIZE) ## Download kustomize locally if necessary.
$(KUSTOMIZE): $(LOCALBIN)
	test -s $(LOCALBIN)/kustomize || { curl -s $(KUSTOMIZE_INSTALL_SCRIPT) | bash -s -- $(subst v,,$(KUSTOMIZE_VERSION)) $(LOCALBIN); }
