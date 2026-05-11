package wgkey

import (
	"crypto/rand"
	"encoding/base64"

	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

func GenerateKeypair() (privateKey, publicKey string, err error) {
	k, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		return "", "", err
	}
	pub := k.PublicKey()
	priv := base64.StdEncoding.EncodeToString(k[:])
	pubS := base64.StdEncoding.EncodeToString(pub[:])
	return priv, pubS, nil
}

func GeneratePSK() (string, error) {
	var b wgtypes.Key
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b[:]), nil
}
