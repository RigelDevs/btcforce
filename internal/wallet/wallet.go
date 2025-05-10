// internal/wallet/wallet.go
package wallet

import (
	"fmt"
	"math/big"
	"os"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
)

type WalletInfo struct {
	Address    string
	WIF        string
	PrivateKey string
}

func FromPrivateKey(privKey *big.Int) *WalletInfo {
	// Convert big.Int to 32-byte array
	bytes := privKey.Bytes()
	if len(bytes) > 32 {
		return nil
	}

	// Pad with zeros if necessary
	paddedBytes := make([]byte, 32)
	copy(paddedBytes[32-len(bytes):], bytes)

	// Create private key
	privateKey, _ := btcec.PrivKeyFromBytes(paddedBytes)
	if privateKey == nil {
		return nil
	}

	// Get public key
	publicKey := privateKey.PubKey()

	// Create P2PKH address using btcutil.Hash160
	// This internally uses SHA-256 + RIPEMD-160 as required by Bitcoin
	pubKeyHash := btcutil.Hash160(publicKey.SerializeCompressed())
	address, err := btcutil.NewAddressPubKeyHash(pubKeyHash, &chaincfg.MainNetParams)
	if err != nil {
		return nil
	}

	// Create WIF
	wif, err := btcutil.NewWIF(privateKey, &chaincfg.MainNetParams, true)
	if err != nil {
		return nil
	}

	return &WalletInfo{
		Address:    address.EncodeAddress(),
		WIF:        wif.String(),
		PrivateKey: fmt.Sprintf("%064x", privKey),
	}
}

// FromPrivateKeyHex creates a wallet from a hex string private key
func FromPrivateKeyHex(hexKey string) *WalletInfo {
	privKey := new(big.Int)
	privKey.SetString(hexKey, 16)
	return FromPrivateKey(privKey)
}

func LogFound(msg string) error {
	file, err := os.OpenFile("wallets_found.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteString(msg)
	return err
}
