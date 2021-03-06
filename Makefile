GO_BUILD_ENV := CGO_ENABLED=0 GOOS=linux
OUT_DIR=$(shell pwd)/bin
OUT_FILE=$(OUT_DIR)/`basename $(PWD)`
DOCKER_ID=`docker build -q .`

all:
	mkdir -p $(OUT_DIR)
	$(GO_BUILD_ENV) go build -v -o $(OUT_FILE) .

arm:
	mkdir -p $(OUT_DIR)
	$(GO_BUILD_ENV) GOARCH=arm go build -v -o $(OUT_FILE) .

docker:
	docker build -t iuyte/xkcd . --build-arg TOKEN='$(DISCORD_TOKEN)' --build-arg YOUTUBE_TOKEN='$(YOUTUBE_TOKEN)'

run:
	docker run iuyte/xkcd

clean:
	rm -rf $(OUT_DIR)
