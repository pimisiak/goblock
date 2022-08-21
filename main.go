package main

import (
	"fmt"
	"github.com/pimisiak/goblock/blockchain"
)

func main() {
  chain := blockchain.InitBlockchain()

  chain.AddBlock("First Block")
  chain.AddBlock("Second Block")
  chain.AddBlock("Third Block")

  for _, block := range chain.Blocks {
    fmt.Printf("Previous Hash: %x\n", block.PrevHash)
    fmt.Printf("Data in Block: %s\n", block.Data)
    fmt.Printf("Hash: %x\n", block.Hash)
  }
}
