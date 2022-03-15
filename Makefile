.PHONY: all build-binary build-image push clean

all: build-binary

TAG ?= 0.0.1
REGISTRY ?= vrgf2003
APP_NAME ?= simple-apiserver


build-binary:
	go build -o ${APP_NAME} .


build-image: build-binary
	docker build -t ${REGISTRY}/${APP_NAME}:${TAG} .


push: build-image
	docker push ${REGISTRY}/${APP_NAME}:${TAG}

clean:
	@echo Cleaning up...
	rm -rf ${APP_NAME}
