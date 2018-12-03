all: deps

# example:
# $ make run PEER="/ip4/127.0.0.1/tcp/3330/ipfs/QmZPNaCnnR59Dtw5nUuxv33pNXxRqKurnZTHLNJ6LaqEnx" IMAGE="QmSHfs2RyGYZY7fnzuBGAyKTe6Ve4jm5PBjHx3mPa42My3"
.PHONY: run
run:
	@go run main.go -peer="$(PEER)" -image="$(IMAGE)"

# example:
# $ make run/genesis PEER="/ip4/127.0.0.1/tcp/3330/ipfs/QmZPNaCnnR59Dtw5nUuxv33pNXxRqKurnZTHLNJ6LaqEnx" IMAGE="QmSHfs2RyGYZY7fnzuBGAyKTe6Ve4jm5PBjHx3mPa42My3"
.PHONY: run/genesis
run/genesis:
	@go run main.go -peer="$(PEER)" -genesis=true -image="$(IMAGE)"

.PHONY: deps
deps:
	@rm -rf ./vendor && \
	echo "running dep ensure..." && \
	dep ensure -v && \
	$(MAKE) gxundo

.PHONY: gxundo
gxundo:
	@bash scripts/gxundo.sh vendor/

.PHONY: install/gxundo
install/gxundo:
	@wget https://raw.githubusercontent.com/c3systems/gxundo/master/gxundo.sh \
	-O scripts/gxundo.sh && \
	chmod +x scripts/gxundo.sh

# NOTE: Temp fix till PR is merged:
# https://github.com/libp2p/go-libp2p-crypto/pull/35
.PHONY: fix/libp2pcrypto
fix/libp2pcrypto:
	@rm -rf vendor/github.com/libp2p/go-libp2p-crypto/
	@git clone -b forPR https://github.com/c3systems/go-libp2p-crypto.git
	@mv go-libp2p-crypto vendor/github.com/libp2p/go-libp2p-crypto
	@find "./vendor" -name "*.go" -print0 | xargs -0 perl -pi -e "s/c3systems\/go-libp2p-crypto/libp2p\/go-libp2p-crypto/g"
	#@sed -iE 's/k1, k2 :=/k1, k2, _ :=/g' vendor/github.com/libp2p/go-libp2p-secio/protocol.go
	#@sed -iE 's/s.local.keys = k1/\/\/s.local.keys = k1/g' vendor/github.com/libp2p/go-libp2p-secio/protocol.go
	#@sed -iE 's/s.remote.keys = k2/\/\/s.remote.keys = k2/g' vendor/github.com/libp2p/go-libp2p-secio/protocol.go

  @git clone git@github.com:gogo/protobuf.git
	@rm -rf vendor/github.com/gogo/protobuf
	@mv protobuf vendor/github.com/gogo/

	@git clone git@github.com:libp2p/go-libp2p-netutil.git
	@rm -rf vendor/github.com/libp2p/go-libp2p-netutil
	@mv go-libp2p-netutil vendor/github.com/libp2p/
