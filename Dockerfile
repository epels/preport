FROM golang:1.16-alpine
COPY . /src
WORKDIR /src/github.com/epels/preport
RUN go build -o /bin/preport github.com/epels/preport/cmd/preport
CMD ["/bin/preport"]
