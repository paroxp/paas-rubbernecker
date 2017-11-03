RUBBERNECKER_DIR=$(shell pwd)

test:
	docker-compose run rubbernecker go test -v ./...

sass:
	docker run --rm -v ${RUBBERNECKER_DIR}:${RUBBERNECKER_DIR} -w ${RUBBERNECKER_DIR} ubuntudesign/sass sass pkg/rubbernecker/styles/app.scss > pkg/rubbernecker/styles/app.css --style compressed --no-cache
