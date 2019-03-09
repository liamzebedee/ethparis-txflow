package main

import (
	"encoding/json"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/ethdb"
	"io/ioutil"
	"log"
	"strings"
)

var db ethdb.Database
var cfg Config

func init() {
	log.SetFlags(0)
	var err error
	db, err = rawdb.NewLevelDBDatabase("datadir/geth/chaindata", 1024, 1024, "")
	if err != nil {
		log.Fatal(err)
	}

	data, err := ioutil.ReadFile("config.json")
	if err != nil {
		log.Fatal(err)
	}

	err = json.Unmarshal(data, &cfg)
	if err != nil {
		log.Fatal(err)
	}
}

type Config struct {
	Tx   string `json:"tx"`
	From string `json:"from"`
}

type Chain struct {
	Header *types.Header
}

func (c *Chain) Engine() consensus.Engine {
	panic("implement me")
}

func (c *Chain) GetHeader(hash common.Hash, number uint64) *types.Header {
	return rawdb.ReadHeader(db, hash, number)
}

func main() {
	txHash := common.HexToHash(cfg.Tx)
	from := common.HexToAddress(cfg.From)

	tx, blockHash, blockNumber, _ := rawdb.ReadTransaction(db, txHash)

	block := rawdb.ReadBlock(db, blockHash, blockNumber)

	chainCfg := rawdb.ReadChainConfig(db, rawdb.ReadCanonicalHash(db, 0))

	stateDB, err := state.New(block.Root(), state.NewDatabase(db))
	if err != nil {
		log.Fatal(err)
	}

	message := types.NewMessage(from, tx.To(), 0, tx.Value(), tx.Gas(),
		tx.GasPrice(), tx.Data(), false)

	author := block.Coinbase()

	vmCtx := core.NewEVMContext(message, block.Header(), &Chain{}, &author)

	contracts, err := Contracts()
	if err != nil {
		log.Fatalf("failed parsing contracts")
	}

	tracer := NewTracer(contracts)
	vmConfig := vm.Config{Debug: true, Tracer: tracer}

	env := vm.NewEVM(vmCtx, stateDB, chainCfg, vmConfig)
	_, _, err = env.Call(vm.AccountRef(from), *tx.To(), tx.Data(), tx.Gas(), tx.Value())
	if err != nil {
		log.Fatalf("failed calling contract: %s", err)
	}

	//log.Printf("Result: %x\n", res)

	for _, frame := range tracer.Stack {
		//log.Printf("Depth: %d, Contract: %s, Instruction: %d // %s", frame.Depth, frame.Contract, frame.Instruction, frame.Source)
		contract := contracts[frame.Contract]

		log.Printf("%s:%d%s%s", contract.Name, frame.Line, strings.Repeat("\t", int(frame.Depth+2)), frame.Source)
	}
}
