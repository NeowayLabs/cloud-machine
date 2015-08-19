CGO_ENABLED=0 go build -v -a -installsuffix cgo machine-up.go
CGO_ENABLED=0 go build -v -a -installsuffix cgo cluster-up.go
