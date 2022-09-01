package blockchain

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"log"
	"math/big"
	"strings"

	"github.com/pimisiak/goblock/utils"
	"github.com/pimisiak/goblock/wallet"
)

type Transaction struct {
	ID      []byte
	Inputs  []TxInput
	Outputs []TxOutput
}

func (tx *Transaction) Serialize() []byte {
	var encoded bytes.Buffer
	enc := gob.NewEncoder(&encoded)
	err := enc.Encode(tx)
	utils.Handle(err)
	return encoded.Bytes()
}

func (tx *Transaction) Hash() []byte {
	var hash [32]byte

	txCopy := *tx
	txCopy.ID = []byte{}

	hash = sha256.Sum256(txCopy.Serialize())
	return hash[:]
}

func (tx *Transaction) SetID() {
	var encoded bytes.Buffer
	var hash [32]byte

	encode := gob.NewEncoder(&encoded)
	err := encode.Encode(tx)
	utils.Handle(err)

	hash = sha256.Sum256(encoded.Bytes())
	tx.ID = hash[:]
}

func NewTransaction(from, to string, amount int, u *UTXOSet) *Transaction {
	var inputs []TxInput
	var outputs []TxOutput

	wallets, err := wallet.CreateWallets()
	utils.Handle(err)
	w := wallets.GetWallet(from)
	pubKeyHash := wallet.PublicKeyHash(w.PublicKey)

	acc, validOutputs := u.FindSpendableOutputs(pubKeyHash, amount)

	if acc < amount {
		log.Panic("Error: not enough funds")
	}

	for txid, outs := range validOutputs {
		txID, err := hex.DecodeString(txid)
		utils.Handle(err)
		for _, out := range outs {
			input := TxInput{txID, out, nil, w.PublicKey}
			inputs = append(inputs, input)
		}
	}

	outputs = append(outputs, *NewTXOutput(amount, to))

	if acc > amount {
		outputs = append(outputs, *NewTXOutput(acc-amount, from))
	}

	tx := Transaction{nil, inputs, outputs}
	tx.ID = tx.Hash()
	u.Blockchain.SignTransaction(&tx, w.PrivateKey)

	return &tx
}

func CoinbaseTx(to, data string) *Transaction {
	if data == "" {
		data = fmt.Sprintf("Coins to %s", to)
	}
	txin := TxInput{[]byte{}, -1, nil, []byte(data)}
	txout := NewTXOutput(100, to)

	tx := Transaction{nil, []TxInput{txin}, []TxOutput{*txout}}
	tx.SetID()

	return &tx
}

func (tx *Transaction) isCoinbaseTx() bool {
	return len(tx.Inputs) == 1 && len(tx.Inputs[0].ID) == 0 && tx.Inputs[0].Out == -1
}

func (tx *Transaction) TrimmedCopy() Transaction {
	var inputs []TxInput
	var outputs []TxOutput

	for _, in := range tx.Inputs {
		inputs = append(inputs, TxInput{in.ID, in.Out, nil, nil})
	}

	for _, out := range tx.Outputs {
		outputs = append(outputs, TxOutput{out.Value, out.PubKeyHash})
	}

	txCopy := Transaction{tx.ID, inputs, outputs}
	return txCopy
}

func (tx *Transaction) Verify(prevTxs map[string]Transaction) bool {
	if tx.isCoinbaseTx() {
		return true
	}

	for _, in := range tx.Inputs {
		if prevTxs[hex.EncodeToString(in.ID)].ID == nil {
			log.Panic("ERRO: Previous transaction does not exist")
		}
	}

	txCopy := tx.TrimmedCopy()
	curve := elliptic.P256()

	for inID, in := range txCopy.Inputs {
		prevTx := prevTxs[hex.EncodeToString(in.ID)]
		txCopy.Inputs[inID].Signature = nil
		txCopy.Inputs[inID].PubKey = prevTx.Outputs[in.Out].PubKeyHash
		txCopy.ID = txCopy.Hash()
		txCopy.Inputs[inID].PubKey = nil

		r := big.Int{}
		s := big.Int{}
		sigLen := len(in.Signature)
		r.SetBytes(in.Signature[:(sigLen / 2)])
		s.SetBytes(in.Signature[(sigLen / 2):])

		x := big.Int{}
		y := big.Int{}
		keyLen := len(in.PubKey)
		x.SetBytes(in.PubKey[:(keyLen / 2)])
		y.SetBytes(in.PubKey[(keyLen / 2):])

		rawPubKey := ecdsa.PublicKey{curve, &x, &y}
		if ecdsa.Verify(&rawPubKey, txCopy.ID, &r, &s) == false {
			return false
		}
	}

	return true
}

func (tx *Transaction) Sign(privKey ecdsa.PrivateKey, prevTxs map[string]Transaction) {
	if tx.isCoinbaseTx() {
		return
	}

	for _, in := range tx.Inputs {
		if prevTxs[hex.EncodeToString(in.ID)].ID == nil {
			log.Panic("ERROR: Previous transaction does not exist")
		}
	}

	txCopy := tx.TrimmedCopy()

	for inID, in := range txCopy.Inputs {
		prevTx := prevTxs[hex.EncodeToString(in.ID)]
		txCopy.Inputs[inID].Signature = nil
		txCopy.Inputs[inID].PubKey = prevTx.Outputs[in.Out].PubKeyHash
		txCopy.ID = txCopy.Hash()
		txCopy.Inputs[inID].PubKey = nil

		r, s, err := ecdsa.Sign(rand.Reader, &privKey, txCopy.ID)
		utils.Handle(err)
		signature := append(r.Bytes(), s.Bytes()...)
		tx.Inputs[inID].Signature = signature
	}
}

func (tx *Transaction) String() string {
	var lines []string
	lines = append(lines, fmt.Sprintf("--- Transaction %x:", tx.ID))
	for i, input := range tx.Inputs {
		lines = append(lines, fmt.Sprintf("     Input %d", i))
		lines = append(lines, fmt.Sprintf("       TxID:       %x", input.ID))
		lines = append(lines, fmt.Sprintf("       Out:        %d", input.Out))
		lines = append(lines, fmt.Sprintf("       Signature:  %x", input.Signature))
		lines = append(lines, fmt.Sprintf("       PubKey:     %x", input.PubKey))
	}

	for i, output := range tx.Outputs {
		lines = append(lines, fmt.Sprintf("     Output %d", i))
		lines = append(lines, fmt.Sprintf("       Value:      %d", output.Value))
		lines = append(lines, fmt.Sprintf("       PubKeyHash: %x", output.PubKeyHash))
	}

	return strings.Join(lines, "\n")
}
