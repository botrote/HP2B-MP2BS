peer:
	@go build -o pr_node peerNode.go

clean:
	@if [ -e go.mod ]; then\
		rm go.mod;\
	fi
	@if [ -e go.sum ]; then\
		rm go.sum;\
	fi
	@if [ -e pr_node ]; then \
		rm pr_node;\
	fi

mod :
	$(shell export GO111MODULE=on)
	$(shell go mod init "github.com/Seo-A-Nam/MCNL")
	$(shell go mod tidy -go=1.16 && go mod tidy -go=1.17)

.PHONY: peer