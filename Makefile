project_name = lookingVps
main_path = ./cmd/app

build: 
	@go build -o $(main_path)/$(project_name) $(main_path)/main.go $(main_path)/saver.go
