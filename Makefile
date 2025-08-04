BINARY_NAME = container
SOURCE_FILES = main.go config.go filesystem.go network.go cgroups.go utils.go container.go

.PHONY: build clean

build:
	go build -o $(BINARY_NAME) $(SOURCE_FILES)

clean:
	rm -f $(BINARY_NAME)