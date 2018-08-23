.PHONY: build
#DOCKER_BUILD := docker build . -t builder
#DOCKER_CMD := docker run --rm -v "$(PWD)":/go/src/github.com/enxebre/cluster-api-provider-libvirt:Z -w /go/src/github.com/enxebre/cluster-api-provider-libvirt builder
DOCKER_IMAGE := docker build . -t quay.io/alberto_lamela/libvirt-actuator

#build: ## Build binary
#	@echo -e "\033[32mBuilding package...\033[0m"
#	mkdir -p bin
#	$(DOCKER_CMD) env CGO_ENABLED=1 go build -v -o libvirt-actuator ./

image: ## Build binary
	@echo -e "\033[32mBuilding package...\033[0m"
	mkdir -p bin
	$(DOCKER_IMAGE)

gendeepcopy:
	#go build -o $$GOPATH/bin/deepcopy-gen github.com/enxebre/cluster-api-provider-libvirt/vendor/k8s.io/code-generator/cmd/deepcopy-gen
	deepcopy-gen \
	  -i ./cloud/libvirt/providerconfig,./cloud/libvirt/providerconfig/v1alpha1 \
	  -O zz_generated.deepcopy \
	  -h boilerplate.go.txt