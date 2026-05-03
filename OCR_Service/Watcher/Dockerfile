############################################
# Stage 1: Compile the binary in a containerized Golang environment
############################################
FROM docker.io/golang:latest as build

WORKDIR /app

# Copy the source files from the host
COPY . .

# Set the working directory to the same place we copied the code
WORKDIR /app/OCR_Service/Watcher

ENV GOWORK=off

RUN go mod download

# Build the binary!
RUN CGO_ENABLED=0 GOOS=linux  go build -a -o ocr_watcher

############################################
# Stage 2: Build the container image proper
############################################
FROM alpine:3.20

# Copy the binary from the host.
COPY --from=build /app/OCR_Service/Watcher/ocr_watcher .

# Tell Docker to execute this command on a `docker run`.
ENTRYPOINT ["/ocr_watcher"]