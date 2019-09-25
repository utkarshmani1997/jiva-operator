# Base image for csi service build.
JIVA_OPERATOR_IMAGE ?= utkarshmani1997/jiva-operator:ci
JIVA_CSI_IMAGE ?= utkarshmani1997/jiva-csi:ci

# Output registry and image names for csi plugin image
REGISTRY ?= utkarshmani1997

# Output plugin name and its image name and tag
PLUGIN_NAME=jiva-csi
PLUGIN_TAG=dev
OPERATOR_NAME=jiva-operator
OPERATOR_TAG=dev

GIT_BRANCH = $(shell git rev-parse --abbrev-ref HEAD | sed -e "s/.*\\///")
GIT_TAG = $(shell git describe --tags)

# use git branch as default version if not set by env variable, if HEAD is detached that use the most recent tag
VERSION ?= $(if $(subst HEAD,,${GIT_BRANCH}),$(GIT_BRANCH),$(GIT_TAG))
COMMIT ?= $(shell git rev-parse HEAD | cut -c 1-7)
DATETIME ?= $(shell date +'%F_%T')
BUILD_META ?= unreleased
LDFLAGS ?= \
        -extldflags "-static" \
	-X github.com/utkarshmani1997/jiva-operator/version/version.Version=${VERSION} \
	-X github.com/utkarshmani1997/jiva-operator/version/version.Commit=${COMMIT} \
	-X github.com/utkarshmani1997/jiva-operator/version/version.DateTime=${DATETIME} \
	-X github.com/utkarshmani1997/jiva-operator/version/version.BuildMeta=${BUILD_META}

IMAGE_TAG ?= dev
REGISTRY_PATH=${REGISTRY}/${PLUGIN_NAME}:${PLUGIN_TAG}

.PHONY: all
all:
	@echo "Available commands:"
	@echo "  build-operator                           - build operator source code"
	@echo "  container-operator                       - build operator container"
	@echo "  push-operator                            - push operator to dockerhub registry (${REGISTRY})"
	@echo "  build-plugin                             - build operator source code"
	@echo "  container-plugin                         - build operator container"
	@echo "  push-plugin                              - push operator to dockerhub registry (${REGISTRY})"
	@echo ""
	@make print-variables --no-print-directory

.PHONY: print-variables
print-variables:
	@echo "Variables:"
	@echo "  VERSION:    ${VERSION}"
	@echo "  GIT_BRANCH: ${GIT_BRANCH}"
	@echo "  GIT_TAG:    ${GIT_TAG}"
	@echo "  COMMIT:     ${COMMIT}"
	@echo "Testing variables:"
	@echo " Produced Image: ${JIVA_CSI_IMAGE}, ${JIVA_OPERATOR_IMAGE}"
	@echo " REGISTRY: ${REGISTRY}"


.get:
	rm -rf ./build/_output/
	GO111MODULE=on go mod download

build-operator: .get
	GO111MODULE=on GOOS=linux go build -a -ldflags '$(LDFLAGS)' -o ./build/_output/$(OPERATOR_NAME) ./cmd/manager/main.go

build-plugin: .get
	GO111MODULE=on GOOS=linux go build -a -ldflags '$(LDFLAGS)' -o ./build/_output/$(PLUGIN_NAME) ./cmd/csi/main.go

container-plugin: build-plugin
	docker build -f ./build/Dockerfile.plugin -t $(REGISTRY)/$(PLUGIN_NAME):$(PLUGIN_TAG) .

container-operator: build-operator
	docker build -f Dockerfile -t $(REGISTRY)/$(OPERATOR_NAME):$(OPERATOR_TAG) .

generate:
	operator-sdk generate k8s --verbose

operator:
	operator-sdk build $(REGISTRY)/$(OPERATOR_NAME):$(OPERATOR_TAG) --verbose

push-csi: container-plugin
	docker push $(REGISTRY)/$(PLUGIN_NAME):$(PLUGIN_TAG)

push-operator: container-operator
	docker push $(REGISTRY)/$(OPERATOR_NAME):$(OPERATOR_TAG)

clean: .get
