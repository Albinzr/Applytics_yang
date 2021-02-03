# Making build for project
FROM golang:alpine AS goBuilder
WORKDIR /go/src/yang
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags '-w -s' -a -installsuffix cgo -o yang

# Running project with the build
FROM alpine:latest  
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=goBuilder /go/src/yang/yang /go/src/yang/production.env /go/src/yang/local.env ./
# RUN mkdir temp
# RUN cat > temp.txt
# RUN ls && pwd
RUN echo "....................................................................*......................................."
CMD ["./yang","-env","production"]  

