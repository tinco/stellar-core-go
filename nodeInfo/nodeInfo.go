package nodeInfo

import (
	"crypto/sha256"
	"fmt"

	"github.com/stellar/go/strkey"
	"github.com/stellar/go/xdr"
	"golang.org/x/crypto/ed25519"
)

const SecretSeedString string = "SAN6S4HURKTECO6MGKDKNPQUZFEDDW7CODR63ZIEKGFW27MUWZX2TNV2"
const PublicKeyString string = "GCFVEVUGA62TM3P2HCBZRRAGIV4CMDNZGQYXD733LSEO6RDHPU5H7MOX"
const NetworkPassPhrase string = "Public Global Stellar Network ; September 2015"

type NodeInfo struct {
	SecretSeed xdr.Uint256
	PublicKey  xdr.Uint256
	PrivateKey ed25519.PrivateKey
	NetworkID  xdr.Hash
}

func SetupCrypto() NodeInfo {
	var nodeInfo NodeInfo
	secretSeedBytes, err := strkey.Decode(strkey.VersionByteSeed, SecretSeedString)
	if err != nil {
		fmt.Println(err)
		panic("Could not initialize keys from seed.")
	}

	publicKeyBytes, err := strkey.Decode(strkey.VersionByteAccountID, PublicKeyString)
	if err != nil {
		fmt.Println(err)
		panic("Could not initialize keys from public key.")
	}

	copy(nodeInfo.SecretSeed[:], secretSeedBytes)
	copy(nodeInfo.PublicKey[:], publicKeyBytes)

	nodeInfo.NetworkID = xdr.Hash(sha256.Sum256([]byte(NetworkPassPhrase)))
	nodeInfo.PrivateKey = ed25519.NewKeyFromSeed(secretSeedBytes)

	return nodeInfo
}
