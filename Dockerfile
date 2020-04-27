FROM golang:1.14-buster AS golang-ffprobe

# Install ffmpeg containing ffprobe
RUN apt-get update && apt-get install -y --no-install-recommends \
		ffmpeg \
	&& rm -rf /var/lib/apt/lists/*

