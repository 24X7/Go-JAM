# Use the official Golang image to create a build artifact.
# This is based on Debian and sets the GOPATH to /go.
# https://hub.docker.com/_/golang
FROM prantlf/golang-make-nodejs-git:1.15-lts as go-build

# Create and change to the app directory.
WORKDIR /.
COPY *.go ./
COPY **/*.go ./**/
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -mod=readonly -v -o main

FROM node:17-alpine as build-step
RUN mkdir -p /apps/honu-works-console
WORKDIR /apps/honu-works-console
COPY /applications/honu-works-console/package.json /app
RUN npm install
COPY /applications/honu-works-console/. /app
RUN npm run build --prod

# Use the official Alpine image for a lean production container.
# https://hub.docker.com/_/alpine
# https://docs.docker.com/develop/develop-images/multistage-build/#use-multi-stage-builds
FROM alpine:3
RUN apk add --no-cache ca-certificates

# Copy the binary to the production image from the builder stage.
COPY --from=go-build /app/server /server
COPY index.html ./index.html
COPY assets/ ./assets/

# Run the web service on container startup.
CMD [""]
