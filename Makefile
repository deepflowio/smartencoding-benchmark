MESSAGE = github.com/deepflowys/deepflow/message
LIBS = github.com/deepflowys/deepflow/server/libs
SERVER = github.com/deepflowys/deepflow/server
FLAGS = -gcflags "-l -l" 
BINARY_SUFFIX :=

.PHONY: all
all: clean cktest

vendor:
	go mod tidy && go mod download && go mod vendor
	cp -r ./message/* vendor/${MESSAGE}/
	find vendor -type d -exec chmod +w {} \;
	cp vendor/${MESSAGE}/metric.proto vendor/${LIBS}/zerodoc/pb
	cp vendor/${MESSAGE}/flow_log.proto vendor/${LIBS}/datatype/pb
	cp vendor/${MESSAGE}/stats.proto vendor/${LIBS}/stats/pb
	cd vendor/${LIBS} && go mod init libs && go generate ./...
	cd vendor/${MESSAGE} &&  go generate ./...

.PHONY: cli
cktest: vendor
	go build -mod vendor ${FLAGS} -o cktest main.go

.PHONY: clean
clean:
	touch vendor
	chmod -R 777 vendor
	rm -rf vendor
	rm -rf cktest
