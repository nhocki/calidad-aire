GO?=go
ZIP?=zip

OUTPUT?=scrapper
LAMBDA_NAME?=siataScrapper
ZIP_FILE?=scrapper.zip

.PHONY: static

default: build

build:
	GOOS=linux GOARCH=amd64 ${GO} build -o ${OUTPUT} ./cmd/worker/main.go

clean:
	rm ${OUTPUT} ${ZIP_FILE}

zip: build
	${ZIP} -r ${ZIP_FILE} ${OUTPUT}

deploy: zip static
	AWS_REGION=us-east-1 aws lambda update-function-code --function-name ${LAMBDA_NAME} --zip-file fileb://${ZIP_FILE} --publish

static:
	sh ./deploy.sh
