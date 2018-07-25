package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/c3systems/c3-go/core/chain/mainchain"
	"github.com/c3systems/c3-go/core/chain/statechain"
	"github.com/c3systems/c3-go/core/p2p/protobuff"
	nodetypes "github.com/c3systems/c3-go/node/types"
	"github.com/c3systems/c3/common/c3crypto"

	ipfsaddr "github.com/ipfs/go-ipfs-addr"
	csms "github.com/libp2p/go-conn-security-multistream"
	lCrypt "github.com/libp2p/go-libp2p-crypto"
	peer "github.com/libp2p/go-libp2p-peer"
	peerstore "github.com/libp2p/go-libp2p-peerstore"
	secio "github.com/libp2p/go-libp2p-secio"
	swarm "github.com/libp2p/go-libp2p-swarm"
	tptu "github.com/libp2p/go-libp2p-transport-upgrader"
	bhost "github.com/libp2p/go-libp2p/p2p/host/basic"
	tcp "github.com/libp2p/go-tcp-transport"
	ma "github.com/multiformats/go-multiaddr"
	msmux "github.com/whyrusleeping/go-smux-multistream"
	yamux "github.com/whyrusleeping/go-smux-yamux"
)

const (
	imageHash = "foo"
	peerStr   = "baz"
	method    = "processImage"
	uri       = "/ip4/0.0.0.0/tcp/9000"
)

var pBuff *protobuff.Node

func getHeadblock() (mainchain.Block, error) {
	return mainchain.Block{}, nil
}

func broadcastTx(tx *statechain.Transaction) (*nodetypes.SendTxResponse, error) {
	return nil, nil
}

func handler(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	var buf bytes.Buffer
	// note: second field is header
	file, _, err := r.FormFile("file")
	if err != nil {
		fmt.Printf("err getting file %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError)
	}
	defer file.Close()

	// name := strings.Split(header.Filename, ".")
	// fmt.Printf("File name %s\n", name[0])
	// Copy the file data to my buffer
	if _, err = io.Copy(&buf, file); err != nil {
		fmt.Printf("err copying %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError)
	}

	wPriv, wPub, err := lCrypt.GenerateKeyPairWithReader(lCrypt.RSA, 4096, rand.Reader)
	if err != nil {
		fmt.Printf("err generating keypairs %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError)
	}

	pid, err := peer.IDFromPublicKey(wPub)
	if err != nil {
		fmt.Printf("err getting pid %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError)
	}

	listen, err := ma.NewMultiaddr(uri)
	if err != nil {
		fmt.Printf("err listening %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError)
	}

	ps := peerstore.NewPeerstore()
	if err = ps.AddPrivKey(pid, wPriv); err != nil {
		fmt.Printf("err adding priv key %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError)
	}
	if err = ps.AddPubKey(pid, wPub); err != nil {
		fmt.Printf("err adding pub key %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError)
	}

	swarmNet := swarm.NewSwarm(ctx, pid, ps, nil)
	tcpTransport := tcp.NewTCPTransport(genUpgrader(swarmNet))
	if err = swarmNet.AddTransport(tcpTransport); err != nil {
		fmt.Printf("err adding transport %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError)
	}
	if err = swarmNet.AddListenAddr(listen); err != nil {
		fmt.Printf("err adding listenaddr %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError)
	}
	newNode := bhost.New(swarmNet)

	addr, err := ipfsaddr.ParseString(peerStr)
	if err != nil {
		fmt.Printf("err parsing peer string %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError)
	}

	pinfo, err := peerstore.InfoFromP2pAddr(addr.Multiaddr())
	if err != nil {
		fmt.Printf("err getting pinfo %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError)
	}

	log.Println("[node] FULL", addr.String())
	log.Println("[node] PIN INFO", pinfo)

	if err = newNode.Connect(ctx, *pinfo); err != nil {
		fmt.Printf("err connecting to peer %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError)
	}

	newNode.Peerstore().AddAddrs(pinfo.ID, pinfo.Addrs, peerstore.PermanentAddrTTL)

	pBuff, err = protobuff.NewNode(&protobuff.Props{
		Host:                   *newNode,
		GetHeadBlockFN:         getHeadblock,
		BroadcastTransactionFN: broadcastTx,
	})
	if err != nil {
		fmt.Printf("error starting protobuff node\n%v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError)
	}

	priv, pub, err := c3crypto.NewKeyPair()
	if err != nil {
		fmt.Printf("error getting keypair\n%v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError)
	}
	pubAddr, err := c3crypto.EncodeAddress(pub)
	if err != nil {
		fmt.Printf("error getting addr\n%v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError)
	}

	tx := statechain.NewTransaction(&statechain.TransactionProps{
		ImageHash: imageHash,
		Method:    method,
		Payload:   buf.Bytes(),
		From:      pubAddr,
	})
	if err = tx.SetSig(priv); err != nil {
		fmt.Printf("error setting sig\n%v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError)
	}
	if err = tx.SetHash(); err != nil {
		fmt.Printf("error setting hash\n%v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError)
	}
	if tx.Props().TxHash == nil {
		fmt.Print("tx hash is nil!")
		http.Error(w, http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError)
	}

	txBytes, err := tx.Serialize()
	if err != nil {
		fmt.Printf("error getting tx bytes\n%v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError)
	}
	go func() {
		ch := make(chan interface{})
		self := newNode.ID()
		var peer peer.ID
		for _, peerID := range newNode.Peerstore().Peers() {
			if peerID != self {
				peer = peerID
				break
			}
		}
		pBuff.ProcessTransaction.SendTransaction(peer, txBytes, ch)

		res := <-ch
		fmt.Printf("received response on channel %v", res)
	}()

	if _, err = fmt.Fprint(w, *tx.Props().TxHash); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError)
	}
}

func main() {
	http.HandleFunc("/submit", handler)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./index.html")
	})

	log.Println("listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// note: https://github.com/libp2p/go-libp2p-swarm/blob/da01184afe4c67bec58c5e73f3350ad80b624c0d/testing/testing.go#L39
func genUpgrader(n *swarm.Swarm) *tptu.Upgrader {
	id := n.LocalPeer()
	pk := n.Peerstore().PrivKey(id)
	secMuxer := new(csms.SSMuxer)
	secMuxer.AddTransport(secio.ID, &secio.Transport{
		LocalID:    id,
		PrivateKey: pk,
	})

	stMuxer := msmux.NewBlankTransport()
	stMuxer.AddTransport("/yamux/1.0.0", yamux.DefaultTransport)

	return &tptu.Upgrader{
		Secure:  secMuxer,
		Muxer:   stMuxer,
		Filters: n.Filters,
	}
}
