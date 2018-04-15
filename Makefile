all: magneticod magneticow

magneticod:
	go install ./cmd/magneticod

magneticow:
	# TODO: minify files!
	go-bindata -o="cmd/magneticow/bindata.go" -prefix="cmd/magneticow/data/" cmd/magneticow/data/...
	go install ./cmd/magneticow
