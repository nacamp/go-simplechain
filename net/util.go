package net

import (
	"encoding/hex"

	peerstore "github.com/libp2p/go-libp2p-peerstore"

	multiaddr "github.com/multiformats/go-multiaddr"

	crypto "github.com/libp2p/go-libp2p-crypto"
)

/*
	sample privatekey
	//08021220a178bc3f8ee6738af0139d9784519e5aa1cb256c12c54444bd63296502f29e94
	//080212201afa45f64cd5a28cd40e178889ed2e9f987658bc4d48d376ef6ecb1ab1b26211
	//08021220114a228984dea82c6d7e6996a85fccc7ae6053249dbf5aa5698ffb14668d68f4
	//08021220b7d27774a2671c280d12c15878ecdc0ff8917704c154782445a25a25d962bae8
*/

func PrivateKeyToHex(privKey crypto.PrivKey) (string, error) {
	// priv, _, _ := crypto.GenerateKeyPair(crypto.Secp256k1, 256)
	b, err := crypto.MarshalPrivateKey(privKey)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func HexToPrivateKey(privKey string) (crypto.PrivKey, error) {
	b, err := hex.DecodeString(privKey)
	if err != nil {
		//errors.New("invalid hex string")
		return nil, err
	}
	privKey2, err := crypto.UnmarshalPrivateKey(b)
	return privKey2, err
}

func AddrFromPeerInfo(info *peerstore.PeerInfo) multiaddr.Multiaddr {
	for _, addr := range info.Addrs {
		//why p2p-circuit ?
		if addr.String() != "/p2p-circuit" {
			return addr
		}
	}
	return nil

}
