package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"os"
	"strings"

	"github.com/DaveAppleton/memorykeys"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/compiler"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
)

type tmplMethod struct {
	Original   abi.Method // Original method as parsed by the abi package
	Normalized abi.Method // Normalized version of the parsed method (capitalized names, non-anonymous args/returns)
	Structured bool       // Whether the returns should be accumulated into a struct
}
type tmplEvent struct {
	Original   abi.Event // Original event as parsed by the abi package
	Normalized abi.Event // Normalized version of the parsed fields
}

func index(w http.ResponseWriter, r *http.Request) {
	ShowFile(w, "index.html")
}

// func DeployToken(auth *bind.TransactOpts, backend bind.ContractBackend, _target common.Address, _lastModification *big.Int, _lastTransfer *big.Int, numTokens *big.Int, name string, symbol string) (common.Address, *types.Transaction, *ZBCToken, error) {
// 	parsed, err := abi.JSON(strings.NewReader(ZBCTokenABI))
// 	if err != nil {
// 		return common.Address{}, nil, nil, err
// 	}

// 	address, tx, contract, err := bind.DeployContract(auth, parsed, common.FromHex(ZBCTokenBin), backend, _target, _lastModification, _lastTransfer, numTokens, name, symbol)
// 	if err != nil {
// 		return common.Address{}, nil, nil, err
// 	}
// 	return address, tx, &ZBCToken{ZBCTokenCaller: ZBCTokenCaller{contract: contract}, ZBCTokenTransactor: ZBCTokenTransactor{contract: contract}, ZBCTokenFilterer: ZBCTokenFilterer{contract: contract}}, nil
// }
var (
	baseClient *backends.SimulatedBackend
	contract   *bind.BoundContract
	address    common.Address
	tx         *types.Transaction
)

func getClient() (client *backends.SimulatedBackend, err error) {
	if baseClient != nil {
		return baseClient, nil
	}
	funds, _ := new(big.Int).SetString("1000000000000000000000000000", 10)
	banker, _ := memorykeys.GetAddress("banker")
	baseClient = backends.NewSimulatedBackend(core.GenesisAlloc{
		*banker: {Balance: funds},
	}, 8000000)
	return baseClient, nil
}

func test(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(32 << 20) // limit your max input length!
	file, header, err := r.FormFile("file")
	if err != nil {
		fmt.Println(err)
		fmt.Fprintln(w, err)
		return
	}
	fmt.Fprintln(w, header.Filename)
	defer file.Close()
	f, err := os.OpenFile("uploads/"+header.Filename, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		fmt.Fprint(w, err)
		return
	}
	defer f.Close()
	io.Copy(f, file)
	tFile, tHeader, err := r.FormFile("test")
	if err != nil {
		fmt.Println(err)
		fmt.Fprintln(w, err)
		return
	}
	defer tFile.Close()

	name := strings.Split(header.Filename, ".")
	fmt.Printf("File name %s\n", name[0])
	s, err := newScript(w, tHeader.Filename, "uploads/"+header.Filename)
	if err != nil {
		fmt.Println(err)
		fmt.Fprintln(w, err)
		return
	}
	err = s.run()
	if err != nil {
		fmt.Println(err)
		fmt.Fprintln(w, err)
		return
	}

}

func loadfile(w http.ResponseWriter, r *http.Request) {
	client, err := getClient()
	if err != nil {
		fmt.Println(err)
		panic(err)
	}
	r.ParseMultipartForm(32 << 20) // limit your max input length!
	file, header, err := r.FormFile("file")
	if err != nil {
		fmt.Println(err)
		panic(err)
	}
	defer file.Close()
	name := strings.Split(header.Filename, ".")
	fmt.Printf("File name %s\n", name[0])

	f, err := os.OpenFile("uploads/"+header.Filename, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		fmt.Fprint(w, err)
		return
	}
	defer f.Close()
	io.Copy(f, file)
	dataMap, err := compiler.CompileSolidity("", "uploads/"+header.Filename)
	if err != nil {
		fmt.Fprint(w, err)
		return
	}
	for key, fileData := range dataMap {
		fmt.Fprintln(w, key)
		// fmt.Fprintln(w, "===== code =======")
		// fmt.Fprintln(w, fileData.Code)
		// fmt.Fprintln(w, "===== runtime =====")
		// fmt.Fprintln(w, fileData.RuntimeCode)
		fmt.Fprintln(w, "==== hashes =======")
		for hk, hd := range fileData.Hashes {
			fmt.Fprintln(w, hk, hd)
		}
		fmt.Fprintln(w, "===== ABI =====")
		fmt.Fprintln(w, fileData.Info.AbiDefinition)

		//abiX, err := json.Marshal(fileData.Info.AbiDefinition)
		//fmt.Fprintln(w, string(abiX))
		abiBin, err := json.Marshal(fileData.Info.AbiDefinition)
		if err != nil {
			fmt.Fprint(w, err)
			return
		}
		parsed, err := abi.JSON(strings.NewReader(string(abiBin)))
		if err != nil {
			fmt.Fprint(w, err)
			return
		}

		// strippedABI := strings.Map(func(r rune) rune {
		// 	if unicode.IsSpace(r) {
		// 		return -1
		// 	}
		// 	return r
		// }, string(abiX))
		//fmt.Println(strippedABI)
		auth, err := memorykeys.GetTransactor("banker")
		if err != nil {
			fmt.Fprint(w, err)
			return
		}
		// var (
		// 	calls     = make(map[string]*tmplMethod)
		// 	transacts = make(map[string]*tmplMethod)
		// 	events    = make(map[string]*tmplEvent)
		// 	fallback  *tmplMethod
		// 	receive   *tmplMethod

		// 	// identifiers are used to detect duplicated identifier of function
		// 	// and event. For all calls, transacts and events, abigen will generate
		// 	// corresponding bindings. However we have to ensure there is no
		// 	// identifier coliision in the bindings of these categories.
		// 	callIdentifiers     = make(map[string]bool)
		// 	transactIdentifiers = make(map[string]bool)
		// 	eventIdentifiers    = make(map[string]bool)
		// )
		for _, m := range parsed.Methods {
			fmt.Fprintln(w, m.Name)
			fmt.Fprintln(w, "constant", m.IsConstant())
			fmt.Fprintln(w, "payable", m.IsPayable())
			fmt.Fprintln(w, "ID", common.Bytes2Hex(m.ID))
			fmt.Fprintln(w, "== inputs ==")
			for _, i := range m.Inputs {
				fmt.Fprintln(w, "== ", i.Name, i.Type.String())
			}
			fmt.Fprintln(w, "== ------ ==")
			fmt.Fprintln(w, "== outputs ==")
			for _, i := range m.Outputs {
				fmt.Fprintln(w, "== ", i.Name, i.Type.String())
			}
			fmt.Fprintln(w, "== ------- ==")
		}
		abiCode, err := parsed.Methods["addCarToRegistry"].Inputs.Pack("Ford", "Prefect", big.NewInt(1965), "CHASSIS123", common.HexToAddress("0x402a632019272842BF03f62338966dd89f633464"))
		if err != nil {
			fmt.Fprintln(w, err)
			return
		}
		var ia []interface{}
		ia = append(ia, "Ford")
		ia = append(ia, "Prefect")
		ia = append(ia, big.NewInt(1965))
		ia = append(ia, "CHASSIS123")
		ia = append(ia, common.HexToAddress("0x402a632019272842BF03f62338966dd89f633464"))
		abiCodePV, err := parsed.Methods["addCarToRegistry"].Inputs.PackValues(ia)
		if err != nil {
			fmt.Fprintln(w, err)
			return
		}

		fmt.Fprintln(w, common.Bytes2Hex(abiCode))
		fmt.Fprintln(w, common.Bytes2Hex(abiCodePV))
		address, tx, contract, err = bind.DeployContract(auth, parsed, common.FromHex(fileData.Code), client)
		if err != nil {
			fmt.Fprint(w, err)
			return
		}
		client.Commit()
		fmt.Fprintln(w, "address : ", address.Hex())
		fmt.Fprintln(w, "hash : ", tx.Hash().Hex())
		rcp, err := client.TransactionReceipt(context.Background(), tx.Hash())
		if err != nil {
			fmt.Fprint(w, err)
			return
		}
		fmt.Fprintln(w, "receipt status ", rcp.Status)
		var owner common.Address
		err = contract.Call(nil, &owner, "mainOwner")
		if err != nil {
			fmt.Fprint(w, err)
			return
		}
		fmt.Fprintln(w, "owner = ", owner.Hex())
		fmt.Fprintln(w, parsed.Methods["mainOwner"].Outputs[0].Type)
		tx, err = contract.Transact(auth, "addCarToRegistry", "Ford", "Prefect", big.NewInt(1965), "CHASSIS123", owner)
		if err != nil {
			fmt.Fprint(w, err)
			return
		}
		client.Commit()
		rcp, err = client.TransactionReceipt(context.Background(), tx.Hash())
		fmt.Fprintln(w, "add car rcp", rcp.Status)

		fmt.Fprintf(w, "Car 0")

		err = contract.Call(nil, &owner, "getOwner", big.NewInt(0))
		if err != nil {
			fmt.Fprint(w, err)
			return
		}
		fmt.Fprintln(w, "car 0 owner = ", owner.Hex())

		var model string
		err = contract.Call(nil, &model, "getModel", big.NewInt(0))
		if err != nil {
			fmt.Fprint(w, err)
			return
		}
		fmt.Fprintln(w, "car 0 model = ", model)

	}
	fmt.Fprintln(w, "done")
}
