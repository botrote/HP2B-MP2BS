val:
	@go build -o val_node validatorNode.go

clean:
	@if [ -e go.mod ]; then\
		rm go.mod;\
	fi
	@if [ -e go.sum ]; then\
		rm go.sum;\
	fi
	@if [ -e val_node ]; then \
		rm val_node;\
	fi

mod :
	$(shell export GO111MODULE=on)
	$(shell go mod init "github.com/Seo-A-Nam/MCNL")
	$(shell go mod tidy -go=1.16 && go mod tidy -go=1.17)

.PHONY: val
