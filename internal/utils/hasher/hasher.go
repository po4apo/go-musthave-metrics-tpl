package hasher

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"

	"go.uber.org/zap"
)

type Hasher struct {
	logger zap.Logger
	key    string
}

func NewHasher(logger zap.Logger, key string) (Hasher, error) {
	return Hasher{logger: logger, key: key}, nil
}

func (h Hasher) SignData(data []byte) (string, error) {
	hash := hmac.New(sha256.New, []byte(h.key))
	_, err := hash.Write(data)
	if err != nil {
		return "", err
	}
	sign := hex.EncodeToString(hash.Sum(nil))
	h.logger.Debug("Data was signed successfully.")
	return sign, nil
}

func (h Hasher) VerifyDataSignature(data, sign string) (bool, error) {
	hash := hmac.New(sha256.New, []byte(h.key))
	_, err := hash.Write([]byte(data))
	if err != nil {
		return false, err
	}
	calculatedSign := hash.Sum(nil)
	recievedSing, err := hex.DecodeString(sign)
	if err != nil {
		return false, err
	}
	ok := hmac.Equal(calculatedSign, recievedSing)
	if !ok {
		h.logger.Warn("Signature verification failed!",
			zap.String("sign", sign),
		)
	}

	return ok, nil
}

func (h Hasher) Enabled() bool {
	enable := h.key != ""
	h.logger.Debug("Enebling hasher is", zap.Bool("Enable", enable))
	return enable
}
