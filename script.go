package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/big"
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

type script struct {
	w               io.Writer
	lineCounter     int
	lines           []string
	state           string
	auth            *bind.TransactOpts
	parsed          abi.ABI
	fileData        compiler.Contract
	client          *backends.SimulatedBackend
	contractAddress common.Address
	contract        *bind.BoundContract
	keyEntries      map[string]string
	varEntries      map[string]interface{}
}

func newScript(wr io.Writer, fName string, contract string) (*script, error) {
	bytesRead, err := ioutil.ReadFile(fName)
	if err != nil {
		return nil, err
	}
	s := new(script)
	s.keyEntries = make(map[string]string)
	s.varEntries = make(map[string]interface{})
	s.w = wr
	data := string(bytesRead)
	s.lines = strings.Split(data, "\n")

	s.auth, err = memorykeys.GetTransactor("banker")
	if err != nil {
		return nil, err
	}
	funds, _ := new(big.Int).SetString("1000000000000000000000000000", 10)
	banker, _ := memorykeys.GetAddress("banker")
	s.client = backends.NewSimulatedBackend(core.GenesisAlloc{
		*banker: {Balance: funds},
	}, 8000000)
		fmt.Println("Compiling ",contract)
	dataMap, err := compiler.CompileSolidity("", contract)
	if err != nil {
		return nil, err
	}
	for key, fileData := range dataMap {
		s.fileData = *fileData
		log.Println(key)
		fmt.Fprintln(wr, fileData.Code)
		abiBin, err := json.Marshal(fileData.Info.AbiDefinition)
		if err != nil {
			return nil, err
		}
		s.parsed, err = abi.JSON(strings.NewReader(string(abiBin)))
		if err != nil {
			return nil, err
		}

		// ignore any more contracts
		return s, nil
	}
	return nil, errors.New("No contracts found")
}

func (s *script) err(e error) {
	fmt.Fprintln(s.w, e)
}

func (s *script) getAddressValue(item string) (*common.Address, error) {
	if item[0] == '=' {
		val, ok := s.keyEntries[item[1:]]
		if ok {
			res, _ := memorykeys.GetAddress(val)
			return res, nil
		}
		aVal, ok := s.varEntries[item[1:]]
		if ok {
			return aVal.(*common.Address), nil
		}
		return nil, errors.New("cannot locate address for " + item[1:])
	}
	// literal value
	aVal := common.HexToAddress(item)
	return &aVal, nil
}

func (s *script) getNumValue(item string) (res *big.Int, err error) {
	if item[0] == '=' {
		aVal, ok := s.varEntries[item[1:]]
		if ok {
			return aVal.(*big.Int), nil
		}
		return nil, errors.New("cannot locate address for " + item[1:])
	}
	// literal value
	var ok bool
	res = new(big.Int)
	res, ok = res.SetString(item, 10)
	if !ok {
		err = errors.New("cannot use " + item)
	}
	return
}

func (s *script) getStringValue(item string) (res string, err error) {
	if item[0] == '=' {
		aVal, ok := s.varEntries[item[1:]]
		if ok {
			return aVal.(string), nil
		}
		return "", errors.New("cannot locate variable " + item[1:])
	}
	// literal value
	return item, nil
}

func (s *script) getValue(item string, dataType abi.Type) (res interface{}, err error) {
	switch dataType.String() {
	case "address":
		return s.getAddressValue(item)
	case "uint256":
		return s.getNumValue(item)
	case "string":
		return s.getStringValue(item)
	}

	return
}

func (s *script) makeParams(function string, params []string) (res []interface{}, err error) {
	for _, str := range params {
		fmt.Println(str)
	}
	if len(s.parsed.Methods[function].Inputs) != len(params) {
		return nil, fmt.Errorf("invalid number of parameters for %s, expected %d, received %d", function, len(s.parsed.Methods[function].Inputs), len(params))
	}
	for pos, item := range params {
		val, err := s.getValue(item, s.parsed.Methods[function].Inputs[pos].Type)
		if err != nil {
			return nil, err
		}
		res = append(res, val)
	}
	return
}

func (s *script) launchContract() (err error) {
	if len(s.fileData.Code) == 0 {
		return errors.New("No code to launch")
	}
	s.contractAddress, tx, s.contract, err = bind.DeployContract(s.auth, s.parsed, common.FromHex(s.fileData.Code), s.client)
	if err != nil {
		fmt.Fprint(s.w, err)
		return
	}
	s.client.Commit()
	rct, err := s.client.TransactionReceipt(context.Background(), tx.Hash())
	if err != nil {
		fmt.Fprint(s.w, err)
		return
	}
	if rct.Status == types.ReceiptStatusFailed {
		fmt.Fprintln(s.w, "Contract launch failed")
		return errors.New("Contract launch failed")
	}
	code, err := s.client.CodeAt(context.Background(), s.contractAddress, nil)
	if err != nil {
		fmt.Fprintln(s.w, err)
		return err
	}
	if len(code) == 0 {
		return errors.New("no code found at " + s.contractAddress.Hex())
	}
	fmt.Fprintln(s.w, "found ", len(code), "bytes of code at", s.contractAddress.Hex())
	return nil
}

func (s *script) setKey(key string) (err error) {
	if _, err = memorykeys.GetAddress(key); err != nil {
		return
	}
	s.keyEntries[key] = key
	return nil
}

func (s *script) setstate(newstate string) (err error) {
	switch newstate {
	case "Accounts":
	case "Contract":
	case "Check":
	case "Read":
	case "Test":
	default:
		err = errors.New("invalid state " + newstate)
	}
	s.state = newstate
	return
}

func (s *script) invalid(line string) {
	fmt.Fprintln(s.w, "invalid commane in (", s.state, ")", line)
}

func (s *script) run() (err error) {

	for {
		fmt.Fprintln(s.w, "line ", s.lineCounter)
		if s.lineCounter == len(s.lines) {
			return
		}

		line := s.lines[s.lineCounter]
		if len(line) == 0 {
			return
		}
		s.lineCounter++
		fmt.Println("processing :", s.state, line)
		if line[0:1] == "*" {
			fmt.Println("set state", line[1:])
			if err := s.setstate(line[1:]); err != nil {
				return err
			}
			continue
		}
		switch s.state {
		case "Accounts":
			if err := s.setKey(line); err != nil {
				return err
			}
		case "Contract":
			if line != "launch" {
				s.invalid(line)
				return
			}
			if err := s.launchContract(); err != nil {
				fmt.Fprintln(s.w, line, err)
				return err
			}
			fmt.Fprintln(s.w, "launched at", s.contractAddress.Hex())
		case "Check":
			var result interface{}
			lineItems := strings.Split(line, ",")
			if !s.parsed.Methods[lineItems[1]].IsConstant() {
				fmt.Fprintln(s.w, lineItems[1], "function not constant")
				return
			}
			m := s.parsed.Methods[lineItems[1]]
			w := m.Outputs[0]
			fmt.Println(w.Type)
			var params []interface{}
			params, err = s.makeParams(lineItems[1], lineItems[2:])
			if err != nil {
				fmt.Fprintln(s.w, lineItems[1], err)
				return err
			}
			err = s.contract.Call(nil, &result, lineItems[1], params...)
			if err != nil {
				fmt.Fprintln(s.w, lineItems[1], err)
				return
			}

			switch s.parsed.Methods[lineItems[1]].Outputs[0].Type.String() {
			case "address":
				testVal, err := s.getAddressValue(lineItems[0])
				if err != nil {
					fmt.Fprintln(s.w, "error in test line", s.lineCounter)
					fmt.Fprintln(s.w, line)
					fmt.Fprintln(s.w, err)
				}
				if *testVal != result.(common.Address) {
					fmt.Fprintln(s.w, "error in line", s.lineCounter+1, "expected", testVal.Hex(), "found", result.(common.Address).Hex())
					fmt.Fprintln(s.w, line)
				}
			case "uint256":
				testVal, err := s.getNumValue(lineItems[0])
				if err != nil {
					fmt.Fprintln(s.w, "error in test line", s.lineCounter)
					fmt.Fprintln(s.w, line)
					fmt.Fprintln(s.w, err)
				}
				if testVal.Cmp(result.(*big.Int)) != 0 {
					fmt.Fprintln(s.w, "error in line", s.lineCounter+1, "expected", testVal.String(), "found", result.(*big.Int).String())
					fmt.Fprintln(s.w, line)
				}
			case "string":
				testVal, err := s.getStringValue(lineItems[0])
				if err != nil {
					fmt.Fprintln(s.w, "error in test line", s.lineCounter)
					fmt.Fprintln(s.w, line)
					fmt.Fprintln(s.w, err)
				}
				if testVal != result.(string) {
					fmt.Fprintln(s.w, "error in line", s.lineCounter+1, "expected", testVal, "found", result.(string))
					fmt.Fprintln(s.w, line)
				}
			}
		case "Read":
			var result interface{}
			lineItems := strings.Split(line, ",")
			if !s.parsed.Methods[lineItems[1]].IsConstant() {
				fmt.Fprintln(s.w, lineItems[1], "function not constant")
				fmt.Fprintln(s.w, line)
				return
			}
			m := s.parsed.Methods[lineItems[1]]
			w := m.Outputs[0]
			fmt.Println(w.Type)
			var params []interface{}
			params, err = s.makeParams(lineItems[1], lineItems[2:])
			if err != nil {
				fmt.Fprintln(s.w, lineItems[1], err)
				fmt.Fprintln(s.w, line)
				return err
			}
			err = s.contract.Call(nil, &result, lineItems[1], params...)
			if err != nil {
				fmt.Fprintln(s.w, lineItems[1], err)
				fmt.Fprintln(s.w, line)
				return
			}
			s.varEntries[lineItems[0]] = result
		case "Test":
			var params []interface{}
			lineItems := strings.Split(line, ",")
			transactor, err := memorykeys.GetTransactor(lineItems[0])
			if err != nil {
				fmt.Fprintln(s.w, line)
				return err
			}
			params, err = s.makeParams(lineItems[1], lineItems[2:])
			if err != nil {
				fmt.Fprintln(s.w, line)
				return err
			}
			tx, err := s.contract.Transact(transactor, lineItems[1], params...)
			if err != nil {
				fmt.Fprintln(s.w, line)
				return err
			}
			s.client.Commit()
			rcp, err := s.client.TransactionReceipt(context.Background(), tx.Hash())
			if rcp.Status != types.ReceiptStatusSuccessful {
				return errors.New("function " + lineItems[1] + " failed/reverted")
			}
		default:
			return errors.New("state : \"" + s.state + "\" does not have a handler")
		}

	}
}
