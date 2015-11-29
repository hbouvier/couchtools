GOCC=go
# To Compile the linux version using docker simply invoke the makefile like this:
#
# make GOCC="docker run --rm -t -v ${GOPATH}:/go hbouvier/go-lang:1.5"
PROJECTNAME=couchtools

all: get-deps fmt macos linux arm test coverage

clean:
	rm -rf coverage.out \
	       ${GOPATH}/pkg/{linux_amd64,darwin_amd64,linux_arm}/github.com/hbouvier/${PROJECTNAME} \
	       ${GOPATH}/bin/{linux_amd64,darwin_amd64,linux_arm}/${PROJECTNAME} \
	       release

build: fmt test
	${GOCC} install github.com/hbouvier/${PROJECTNAME}

fmt:
	${GOCC} fmt github.com/hbouvier/${PROJECTNAME}

test:
	# ${GOCC} test -v -cpu 4 -count 1 -coverprofile=coverage.out github.com/hbouvier/${PROJECTNAME}

coverage:
	# ${GOCC} tool cover -html=coverage.out

get-deps:
	# ${GOCC} get ...

linux:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 ${GOCC} install github.com/hbouvier/${PROJECTNAME}

macos:
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 ${GOCC} install github.com/hbouvier/${PROJECTNAME}
	@mkdir -p ${GOPATH}/bin/darwin_amd64
	@mv ${GOPATH}/bin/${PROJECTNAME} ${GOPATH}/bin/darwin_amd64

arm:
	GOOS=linux GOARCH=arm CGO_ENABLED=0 ${GOCC} install github.com/hbouvier/${PROJECTNAME}

release: linux macos arm
	@mkdir -p release/bin/{linux_amd64,darwin_amd64,linux_arm}
	for i in linux_amd64 darwin_amd64 linux_arm; do cp ${GOPATH}/bin/$${i}/${PROJECTNAME} release/bin/$${i}/${PROJECTNAME} ; done
	COPYFILE_DISABLE=1 tar cvzf release/${PROJECTNAME}.v`cat VERSION`.tgz release/bin
	zip -r release/${PROJECTNAME}.v`cat VERSION`.zip release/bin
