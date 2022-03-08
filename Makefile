project_name = $(notdir $(shell pwd))
main_path = ./cmd/app

build: 
	CGO_ENABLED=0 go build -ldflags "-s -w" -o $(main_path)/$(project_name) $(main_path)/main.go

fmt:
	gofmt -s -w .
	
clean:
	@rm -rf $(main_path)/$(project_name)
