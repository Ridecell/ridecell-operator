
# Image URL to use all building/pushing image targets
IMG ?= controller:latest

all: test manager

ifdef CI
CI_GINKGO_ARGS=--v -compilers 4
else
CI_GINKGO_ARGS=
endif

# Run tests
test: generate fmt vet manifests
	ginkgo --randomizeAllSpecs --randomizeSuites --cover --trace --progress ${GINKGO_ARGS} ${CI_GINKGO_ARGS} -r ./pkg ./cmd
	gover

# Build manager binary
manager: generate fmt vet
	go build -o bin/manager github.com/Ridecell/ridecell-operator/cmd/manager

# Run against the configured Kubernetes cluster in ~/.kube/config
run: generate fmt vet
	go run ./cmd/manager/main.go

# Install CRDs into a cluster
install: manifests
	kubectl apply -f config/crds

# Deploy controller in the configured Kubernetes cluster in ~/.kube/config
deploy: manifests
	kubectl apply -f config/crds
	kustomize build config/default | kubectl apply -f -

# Generate manifests e.g. CRD, RBAC etc.
manifests:
	go run vendor/sigs.k8s.io/controller-tools/cmd/controller-gen/main.go all
	@cp config/crds/* helm/templates/crds/

# Run go fmt against code
fmt:
	go fmt ./pkg/... ./cmd/...

# Run go vet against code
vet:
	go vet ./pkg/... ./cmd/...

# Generate code
generate:
	go generate ./pkg/... ./cmd/...

# Build the docker image
docker-build: test
	docker build . -t ${IMG}
	@echo "updating kustomize image patch file for manager resource"
	sed -i'' -e 's@image: .*@image: '"${IMG}"'@' ./config/default/manager_image_patch.yaml

# Push the docker image
docker-push:
	docker push ${IMG}

# Install tools
tools:
	if ! type dep >/dev/null; then go get github.com/golang/dep/cmd/dep; fi
	go get -u github.com/onsi/ginkgo/ginkgo github.com/modocache/gover github.com/mattn/goveralls github.com/matryer/moq

# Install dependencies
dep: tools
	dep ensure

# Display a coverage report
cover:
	go tool cover -html=gover.coverprofile
