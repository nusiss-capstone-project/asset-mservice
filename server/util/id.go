package util

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

func NewQuoteID() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return fmt.Sprintf("q_%s_%s", time.Now().UTC().Format("20060102"), hex.EncodeToString(b)), nil
}

func NewOrderNo() (string, error) {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return fmt.Sprintf("AO%s%s", time.Now().UTC().Format("20060102"), hex.EncodeToString(b)), nil
}

func NewLedgerNo() (string, error) {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return fmt.Sprintf("AL%s%s", time.Now().UTC().Format("20060102"), hex.EncodeToString(b)), nil
}

func NewFiatTransactionNo() (string, error) {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return fmt.Sprintf("FT%s%s", time.Now().UTC().Format("20060102"), hex.EncodeToString(b)), nil
}
