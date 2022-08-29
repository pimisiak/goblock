package utils

import (
	"github.com/mr-tron/base58"
	"log"
)

func Base58Encode(input []byte) []byte {
	encode := base58.Encode(input)
	return []byte(encode)
}

func Base58Decode(input []byte) []byte {
	decode, err := base58.Decode(string(input[:]))
	Handle(err)
	return decode
}

func Handle(err error) {
	if err != nil {
		log.Panic(err)
	}
}
