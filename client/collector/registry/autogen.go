// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package registry

import (
	"errors"
	"math/big"
	"strings"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

// Reference imports to suppress errors if they are not otherwise used.
var (
	_ = errors.New
	_ = big.NewInt
	_ = strings.NewReader
	_ = ethereum.NotFound
	_ = bind.Bind
	_ = common.Big1
	_ = types.BloomLookup
	_ = event.NewSubscription
	_ = abi.ConvertType
)

// PublicKey is an auto generated low-level Go binding around an user-defined struct.
type PublicKey struct {
	X [32]byte
	Y [32]byte
}

// Signature is an auto generated low-level Go binding around an user-defined struct.
type Signature struct {
	V uint8
	R [32]byte
	S [32]byte
}

// RegistryMetaData contains all meta data concerning the Registry contract.
var RegistryMetaData = &bind.MetaData{
	ABI: "[{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes20\",\"name\":\"beneficiary\",\"type\":\"bytes20\"},{\"indexed\":false,\"internalType\":\"uint32\",\"name\":\"untilRewardEpochId\",\"type\":\"uint32\"}],\"name\":\"BeneficiaryChilled\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"voter\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"uint32\",\"name\":\"rewardEpochId\",\"type\":\"uint32\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"signingPolicyAddress\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"submitAddress\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"submitSignaturesAddress\",\"type\":\"address\"},{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"x\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"y\",\"type\":\"bytes32\"}],\"indexed\":false,\"internalType\":\"structPublicKey\",\"name\":\"publicKey\",\"type\":\"tuple\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"registrationWeight\",\"type\":\"uint256\"},{\"components\":[{\"internalType\":\"uint8\",\"name\":\"v\",\"type\":\"uint8\"},{\"internalType\":\"bytes32\",\"name\":\"r\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"s\",\"type\":\"bytes32\"}],\"indexed\":false,\"internalType\":\"structSignature\",\"name\":\"signature\",\"type\":\"tuple\"}],\"name\":\"VoterRegistered\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"voter\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"uint32\",\"name\":\"rewardEpochId\",\"type\":\"uint32\"}],\"name\":\"VoterRemoved\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"bytes20\",\"name\":\"_beneficiary\",\"type\":\"bytes20\"}],\"name\":\"chilledUntilRewardEpochId\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"_rewardEpochId\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_rewardEpochId\",\"type\":\"uint256\"}],\"name\":\"getNumberOfRegisteredVoters\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_rewardEpochId\",\"type\":\"uint256\"}],\"name\":\"getRegisteredVoters\",\"outputs\":[{\"internalType\":\"address[]\",\"name\":\"\",\"type\":\"address[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_voter\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_rewardEpochId\",\"type\":\"uint256\"}],\"name\":\"isVoterRegistered\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"maxVoters\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_rewardEpochId\",\"type\":\"uint256\"}],\"name\":\"newSigningPolicyInitializationStartBlockNumber\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"publicKeyRequired\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_voter\",\"type\":\"address\"},{\"components\":[{\"internalType\":\"uint8\",\"name\":\"v\",\"type\":\"uint8\"},{\"internalType\":\"bytes32\",\"name\":\"r\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"s\",\"type\":\"bytes32\"}],\"internalType\":\"structSignature\",\"name\":\"_signature\",\"type\":\"tuple\"}],\"name\":\"registerVoter\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
}

// RegistryABI is the input ABI used to generate the binding from.
// Deprecated: Use RegistryMetaData.ABI instead.
var RegistryABI = RegistryMetaData.ABI

// Registry is an auto generated Go binding around an Ethereum contract.
type Registry struct {
	RegistryCaller     // Read-only binding to the contract
	RegistryTransactor // Write-only binding to the contract
	RegistryFilterer   // Log filterer for contract events
}

// RegistryCaller is an auto generated read-only Go binding around an Ethereum contract.
type RegistryCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// RegistryTransactor is an auto generated write-only Go binding around an Ethereum contract.
type RegistryTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// RegistryFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type RegistryFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// RegistrySession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type RegistrySession struct {
	Contract     *Registry         // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// RegistryCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type RegistryCallerSession struct {
	Contract *RegistryCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts   // Call options to use throughout this session
}

// RegistryTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type RegistryTransactorSession struct {
	Contract     *RegistryTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts   // Transaction auth options to use throughout this session
}

// RegistryRaw is an auto generated low-level Go binding around an Ethereum contract.
type RegistryRaw struct {
	Contract *Registry // Generic contract binding to access the raw methods on
}

// RegistryCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type RegistryCallerRaw struct {
	Contract *RegistryCaller // Generic read-only contract binding to access the raw methods on
}

// RegistryTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type RegistryTransactorRaw struct {
	Contract *RegistryTransactor // Generic write-only contract binding to access the raw methods on
}

// NewRegistry creates a new instance of Registry, bound to a specific deployed contract.
func NewRegistry(address common.Address, backend bind.ContractBackend) (*Registry, error) {
	contract, err := bindRegistry(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Registry{RegistryCaller: RegistryCaller{contract: contract}, RegistryTransactor: RegistryTransactor{contract: contract}, RegistryFilterer: RegistryFilterer{contract: contract}}, nil
}

// NewRegistryCaller creates a new read-only instance of Registry, bound to a specific deployed contract.
func NewRegistryCaller(address common.Address, caller bind.ContractCaller) (*RegistryCaller, error) {
	contract, err := bindRegistry(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &RegistryCaller{contract: contract}, nil
}

// NewRegistryTransactor creates a new write-only instance of Registry, bound to a specific deployed contract.
func NewRegistryTransactor(address common.Address, transactor bind.ContractTransactor) (*RegistryTransactor, error) {
	contract, err := bindRegistry(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &RegistryTransactor{contract: contract}, nil
}

// NewRegistryFilterer creates a new log filterer instance of Registry, bound to a specific deployed contract.
func NewRegistryFilterer(address common.Address, filterer bind.ContractFilterer) (*RegistryFilterer, error) {
	contract, err := bindRegistry(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &RegistryFilterer{contract: contract}, nil
}

// bindRegistry binds a generic wrapper to an already deployed contract.
func bindRegistry(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := RegistryMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Registry *RegistryRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Registry.Contract.RegistryCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Registry *RegistryRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Registry.Contract.RegistryTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Registry *RegistryRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Registry.Contract.RegistryTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Registry *RegistryCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Registry.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Registry *RegistryTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Registry.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Registry *RegistryTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Registry.Contract.contract.Transact(opts, method, params...)
}

// ChilledUntilRewardEpochId is a free data retrieval call binding the contract method 0x3c5cb76f.
//
// Solidity: function chilledUntilRewardEpochId(bytes20 _beneficiary) view returns(uint256 _rewardEpochId)
func (_Registry *RegistryCaller) ChilledUntilRewardEpochId(opts *bind.CallOpts, _beneficiary [20]byte) (*big.Int, error) {
	var out []interface{}
	err := _Registry.contract.Call(opts, &out, "chilledUntilRewardEpochId", _beneficiary)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// ChilledUntilRewardEpochId is a free data retrieval call binding the contract method 0x3c5cb76f.
//
// Solidity: function chilledUntilRewardEpochId(bytes20 _beneficiary) view returns(uint256 _rewardEpochId)
func (_Registry *RegistrySession) ChilledUntilRewardEpochId(_beneficiary [20]byte) (*big.Int, error) {
	return _Registry.Contract.ChilledUntilRewardEpochId(&_Registry.CallOpts, _beneficiary)
}

// ChilledUntilRewardEpochId is a free data retrieval call binding the contract method 0x3c5cb76f.
//
// Solidity: function chilledUntilRewardEpochId(bytes20 _beneficiary) view returns(uint256 _rewardEpochId)
func (_Registry *RegistryCallerSession) ChilledUntilRewardEpochId(_beneficiary [20]byte) (*big.Int, error) {
	return _Registry.Contract.ChilledUntilRewardEpochId(&_Registry.CallOpts, _beneficiary)
}

// GetNumberOfRegisteredVoters is a free data retrieval call binding the contract method 0x369e9434.
//
// Solidity: function getNumberOfRegisteredVoters(uint256 _rewardEpochId) view returns(uint256)
func (_Registry *RegistryCaller) GetNumberOfRegisteredVoters(opts *bind.CallOpts, _rewardEpochId *big.Int) (*big.Int, error) {
	var out []interface{}
	err := _Registry.contract.Call(opts, &out, "getNumberOfRegisteredVoters", _rewardEpochId)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetNumberOfRegisteredVoters is a free data retrieval call binding the contract method 0x369e9434.
//
// Solidity: function getNumberOfRegisteredVoters(uint256 _rewardEpochId) view returns(uint256)
func (_Registry *RegistrySession) GetNumberOfRegisteredVoters(_rewardEpochId *big.Int) (*big.Int, error) {
	return _Registry.Contract.GetNumberOfRegisteredVoters(&_Registry.CallOpts, _rewardEpochId)
}

// GetNumberOfRegisteredVoters is a free data retrieval call binding the contract method 0x369e9434.
//
// Solidity: function getNumberOfRegisteredVoters(uint256 _rewardEpochId) view returns(uint256)
func (_Registry *RegistryCallerSession) GetNumberOfRegisteredVoters(_rewardEpochId *big.Int) (*big.Int, error) {
	return _Registry.Contract.GetNumberOfRegisteredVoters(&_Registry.CallOpts, _rewardEpochId)
}

// GetRegisteredVoters is a free data retrieval call binding the contract method 0x457c2e47.
//
// Solidity: function getRegisteredVoters(uint256 _rewardEpochId) view returns(address[])
func (_Registry *RegistryCaller) GetRegisteredVoters(opts *bind.CallOpts, _rewardEpochId *big.Int) ([]common.Address, error) {
	var out []interface{}
	err := _Registry.contract.Call(opts, &out, "getRegisteredVoters", _rewardEpochId)

	if err != nil {
		return *new([]common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new([]common.Address)).(*[]common.Address)

	return out0, err

}

// GetRegisteredVoters is a free data retrieval call binding the contract method 0x457c2e47.
//
// Solidity: function getRegisteredVoters(uint256 _rewardEpochId) view returns(address[])
func (_Registry *RegistrySession) GetRegisteredVoters(_rewardEpochId *big.Int) ([]common.Address, error) {
	return _Registry.Contract.GetRegisteredVoters(&_Registry.CallOpts, _rewardEpochId)
}

// GetRegisteredVoters is a free data retrieval call binding the contract method 0x457c2e47.
//
// Solidity: function getRegisteredVoters(uint256 _rewardEpochId) view returns(address[])
func (_Registry *RegistryCallerSession) GetRegisteredVoters(_rewardEpochId *big.Int) ([]common.Address, error) {
	return _Registry.Contract.GetRegisteredVoters(&_Registry.CallOpts, _rewardEpochId)
}

// IsVoterRegistered is a free data retrieval call binding the contract method 0x4f5a9968.
//
// Solidity: function isVoterRegistered(address _voter, uint256 _rewardEpochId) view returns(bool)
func (_Registry *RegistryCaller) IsVoterRegistered(opts *bind.CallOpts, _voter common.Address, _rewardEpochId *big.Int) (bool, error) {
	var out []interface{}
	err := _Registry.contract.Call(opts, &out, "isVoterRegistered", _voter, _rewardEpochId)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsVoterRegistered is a free data retrieval call binding the contract method 0x4f5a9968.
//
// Solidity: function isVoterRegistered(address _voter, uint256 _rewardEpochId) view returns(bool)
func (_Registry *RegistrySession) IsVoterRegistered(_voter common.Address, _rewardEpochId *big.Int) (bool, error) {
	return _Registry.Contract.IsVoterRegistered(&_Registry.CallOpts, _voter, _rewardEpochId)
}

// IsVoterRegistered is a free data retrieval call binding the contract method 0x4f5a9968.
//
// Solidity: function isVoterRegistered(address _voter, uint256 _rewardEpochId) view returns(bool)
func (_Registry *RegistryCallerSession) IsVoterRegistered(_voter common.Address, _rewardEpochId *big.Int) (bool, error) {
	return _Registry.Contract.IsVoterRegistered(&_Registry.CallOpts, _voter, _rewardEpochId)
}

// MaxVoters is a free data retrieval call binding the contract method 0xd5e50a63.
//
// Solidity: function maxVoters() view returns(uint256)
func (_Registry *RegistryCaller) MaxVoters(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Registry.contract.Call(opts, &out, "maxVoters")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MaxVoters is a free data retrieval call binding the contract method 0xd5e50a63.
//
// Solidity: function maxVoters() view returns(uint256)
func (_Registry *RegistrySession) MaxVoters() (*big.Int, error) {
	return _Registry.Contract.MaxVoters(&_Registry.CallOpts)
}

// MaxVoters is a free data retrieval call binding the contract method 0xd5e50a63.
//
// Solidity: function maxVoters() view returns(uint256)
func (_Registry *RegistryCallerSession) MaxVoters() (*big.Int, error) {
	return _Registry.Contract.MaxVoters(&_Registry.CallOpts)
}

// NewSigningPolicyInitializationStartBlockNumber is a free data retrieval call binding the contract method 0xfff50753.
//
// Solidity: function newSigningPolicyInitializationStartBlockNumber(uint256 _rewardEpochId) view returns(uint256)
func (_Registry *RegistryCaller) NewSigningPolicyInitializationStartBlockNumber(opts *bind.CallOpts, _rewardEpochId *big.Int) (*big.Int, error) {
	var out []interface{}
	err := _Registry.contract.Call(opts, &out, "newSigningPolicyInitializationStartBlockNumber", _rewardEpochId)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// NewSigningPolicyInitializationStartBlockNumber is a free data retrieval call binding the contract method 0xfff50753.
//
// Solidity: function newSigningPolicyInitializationStartBlockNumber(uint256 _rewardEpochId) view returns(uint256)
func (_Registry *RegistrySession) NewSigningPolicyInitializationStartBlockNumber(_rewardEpochId *big.Int) (*big.Int, error) {
	return _Registry.Contract.NewSigningPolicyInitializationStartBlockNumber(&_Registry.CallOpts, _rewardEpochId)
}

// NewSigningPolicyInitializationStartBlockNumber is a free data retrieval call binding the contract method 0xfff50753.
//
// Solidity: function newSigningPolicyInitializationStartBlockNumber(uint256 _rewardEpochId) view returns(uint256)
func (_Registry *RegistryCallerSession) NewSigningPolicyInitializationStartBlockNumber(_rewardEpochId *big.Int) (*big.Int, error) {
	return _Registry.Contract.NewSigningPolicyInitializationStartBlockNumber(&_Registry.CallOpts, _rewardEpochId)
}

// PublicKeyRequired is a free data retrieval call binding the contract method 0x92e3e45f.
//
// Solidity: function publicKeyRequired() view returns(bool)
func (_Registry *RegistryCaller) PublicKeyRequired(opts *bind.CallOpts) (bool, error) {
	var out []interface{}
	err := _Registry.contract.Call(opts, &out, "publicKeyRequired")

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// PublicKeyRequired is a free data retrieval call binding the contract method 0x92e3e45f.
//
// Solidity: function publicKeyRequired() view returns(bool)
func (_Registry *RegistrySession) PublicKeyRequired() (bool, error) {
	return _Registry.Contract.PublicKeyRequired(&_Registry.CallOpts)
}

// PublicKeyRequired is a free data retrieval call binding the contract method 0x92e3e45f.
//
// Solidity: function publicKeyRequired() view returns(bool)
func (_Registry *RegistryCallerSession) PublicKeyRequired() (bool, error) {
	return _Registry.Contract.PublicKeyRequired(&_Registry.CallOpts)
}

// RegisterVoter is a paid mutator transaction binding the contract method 0x8f7d0957.
//
// Solidity: function registerVoter(address _voter, (uint8,bytes32,bytes32) _signature) returns()
func (_Registry *RegistryTransactor) RegisterVoter(opts *bind.TransactOpts, _voter common.Address, _signature Signature) (*types.Transaction, error) {
	return _Registry.contract.Transact(opts, "registerVoter", _voter, _signature)
}

// RegisterVoter is a paid mutator transaction binding the contract method 0x8f7d0957.
//
// Solidity: function registerVoter(address _voter, (uint8,bytes32,bytes32) _signature) returns()
func (_Registry *RegistrySession) RegisterVoter(_voter common.Address, _signature Signature) (*types.Transaction, error) {
	return _Registry.Contract.RegisterVoter(&_Registry.TransactOpts, _voter, _signature)
}

// RegisterVoter is a paid mutator transaction binding the contract method 0x8f7d0957.
//
// Solidity: function registerVoter(address _voter, (uint8,bytes32,bytes32) _signature) returns()
func (_Registry *RegistryTransactorSession) RegisterVoter(_voter common.Address, _signature Signature) (*types.Transaction, error) {
	return _Registry.Contract.RegisterVoter(&_Registry.TransactOpts, _voter, _signature)
}

// RegistryBeneficiaryChilledIterator is returned from FilterBeneficiaryChilled and is used to iterate over the raw logs and unpacked data for BeneficiaryChilled events raised by the Registry contract.
type RegistryBeneficiaryChilledIterator struct {
	Event *RegistryBeneficiaryChilled // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *RegistryBeneficiaryChilledIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(RegistryBeneficiaryChilled)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(RegistryBeneficiaryChilled)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *RegistryBeneficiaryChilledIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *RegistryBeneficiaryChilledIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// RegistryBeneficiaryChilled represents a BeneficiaryChilled event raised by the Registry contract.
type RegistryBeneficiaryChilled struct {
	Beneficiary        [20]byte
	UntilRewardEpochId uint32
	Raw                types.Log // Blockchain specific contextual infos
}

// FilterBeneficiaryChilled is a free log retrieval operation binding the contract event 0x23a1b7932916d24f6177b7f7282bb925e3733697d5699c07e0372cd149696345.
//
// Solidity: event BeneficiaryChilled(bytes20 indexed beneficiary, uint32 untilRewardEpochId)
func (_Registry *RegistryFilterer) FilterBeneficiaryChilled(opts *bind.FilterOpts, beneficiary [][20]byte) (*RegistryBeneficiaryChilledIterator, error) {

	var beneficiaryRule []interface{}
	for _, beneficiaryItem := range beneficiary {
		beneficiaryRule = append(beneficiaryRule, beneficiaryItem)
	}

	logs, sub, err := _Registry.contract.FilterLogs(opts, "BeneficiaryChilled", beneficiaryRule)
	if err != nil {
		return nil, err
	}
	return &RegistryBeneficiaryChilledIterator{contract: _Registry.contract, event: "BeneficiaryChilled", logs: logs, sub: sub}, nil
}

// WatchBeneficiaryChilled is a free log subscription operation binding the contract event 0x23a1b7932916d24f6177b7f7282bb925e3733697d5699c07e0372cd149696345.
//
// Solidity: event BeneficiaryChilled(bytes20 indexed beneficiary, uint32 untilRewardEpochId)
func (_Registry *RegistryFilterer) WatchBeneficiaryChilled(opts *bind.WatchOpts, sink chan<- *RegistryBeneficiaryChilled, beneficiary [][20]byte) (event.Subscription, error) {

	var beneficiaryRule []interface{}
	for _, beneficiaryItem := range beneficiary {
		beneficiaryRule = append(beneficiaryRule, beneficiaryItem)
	}

	logs, sub, err := _Registry.contract.WatchLogs(opts, "BeneficiaryChilled", beneficiaryRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(RegistryBeneficiaryChilled)
				if err := _Registry.contract.UnpackLog(event, "BeneficiaryChilled", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseBeneficiaryChilled is a log parse operation binding the contract event 0x23a1b7932916d24f6177b7f7282bb925e3733697d5699c07e0372cd149696345.
//
// Solidity: event BeneficiaryChilled(bytes20 indexed beneficiary, uint32 untilRewardEpochId)
func (_Registry *RegistryFilterer) ParseBeneficiaryChilled(log types.Log) (*RegistryBeneficiaryChilled, error) {
	event := new(RegistryBeneficiaryChilled)
	if err := _Registry.contract.UnpackLog(event, "BeneficiaryChilled", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// RegistryVoterRegisteredIterator is returned from FilterVoterRegistered and is used to iterate over the raw logs and unpacked data for VoterRegistered events raised by the Registry contract.
type RegistryVoterRegisteredIterator struct {
	Event *RegistryVoterRegistered // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *RegistryVoterRegisteredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(RegistryVoterRegistered)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(RegistryVoterRegistered)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *RegistryVoterRegisteredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *RegistryVoterRegisteredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// RegistryVoterRegistered represents a VoterRegistered event raised by the Registry contract.
type RegistryVoterRegistered struct {
	Voter                   common.Address
	RewardEpochId           uint32
	SigningPolicyAddress    common.Address
	SubmitAddress           common.Address
	SubmitSignaturesAddress common.Address
	PublicKey               PublicKey
	RegistrationWeight      *big.Int
	Signature               Signature
	Raw                     types.Log // Blockchain specific contextual infos
}

// FilterVoterRegistered is a free log retrieval operation binding the contract event 0xbfb6cd90b6e2668916d9e034926c84f40bcf94094b0d625ec8eecfdeb2150ae1.
//
// Solidity: event VoterRegistered(address indexed voter, uint32 indexed rewardEpochId, address indexed signingPolicyAddress, address submitAddress, address submitSignaturesAddress, (bytes32,bytes32) publicKey, uint256 registrationWeight, (uint8,bytes32,bytes32) signature)
func (_Registry *RegistryFilterer) FilterVoterRegistered(opts *bind.FilterOpts, voter []common.Address, rewardEpochId []uint32, signingPolicyAddress []common.Address) (*RegistryVoterRegisteredIterator, error) {

	var voterRule []interface{}
	for _, voterItem := range voter {
		voterRule = append(voterRule, voterItem)
	}
	var rewardEpochIdRule []interface{}
	for _, rewardEpochIdItem := range rewardEpochId {
		rewardEpochIdRule = append(rewardEpochIdRule, rewardEpochIdItem)
	}
	var signingPolicyAddressRule []interface{}
	for _, signingPolicyAddressItem := range signingPolicyAddress {
		signingPolicyAddressRule = append(signingPolicyAddressRule, signingPolicyAddressItem)
	}

	logs, sub, err := _Registry.contract.FilterLogs(opts, "VoterRegistered", voterRule, rewardEpochIdRule, signingPolicyAddressRule)
	if err != nil {
		return nil, err
	}
	return &RegistryVoterRegisteredIterator{contract: _Registry.contract, event: "VoterRegistered", logs: logs, sub: sub}, nil
}

// WatchVoterRegistered is a free log subscription operation binding the contract event 0xbfb6cd90b6e2668916d9e034926c84f40bcf94094b0d625ec8eecfdeb2150ae1.
//
// Solidity: event VoterRegistered(address indexed voter, uint32 indexed rewardEpochId, address indexed signingPolicyAddress, address submitAddress, address submitSignaturesAddress, (bytes32,bytes32) publicKey, uint256 registrationWeight, (uint8,bytes32,bytes32) signature)
func (_Registry *RegistryFilterer) WatchVoterRegistered(opts *bind.WatchOpts, sink chan<- *RegistryVoterRegistered, voter []common.Address, rewardEpochId []uint32, signingPolicyAddress []common.Address) (event.Subscription, error) {

	var voterRule []interface{}
	for _, voterItem := range voter {
		voterRule = append(voterRule, voterItem)
	}
	var rewardEpochIdRule []interface{}
	for _, rewardEpochIdItem := range rewardEpochId {
		rewardEpochIdRule = append(rewardEpochIdRule, rewardEpochIdItem)
	}
	var signingPolicyAddressRule []interface{}
	for _, signingPolicyAddressItem := range signingPolicyAddress {
		signingPolicyAddressRule = append(signingPolicyAddressRule, signingPolicyAddressItem)
	}

	logs, sub, err := _Registry.contract.WatchLogs(opts, "VoterRegistered", voterRule, rewardEpochIdRule, signingPolicyAddressRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(RegistryVoterRegistered)
				if err := _Registry.contract.UnpackLog(event, "VoterRegistered", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseVoterRegistered is a log parse operation binding the contract event 0xbfb6cd90b6e2668916d9e034926c84f40bcf94094b0d625ec8eecfdeb2150ae1.
//
// Solidity: event VoterRegistered(address indexed voter, uint32 indexed rewardEpochId, address indexed signingPolicyAddress, address submitAddress, address submitSignaturesAddress, (bytes32,bytes32) publicKey, uint256 registrationWeight, (uint8,bytes32,bytes32) signature)
func (_Registry *RegistryFilterer) ParseVoterRegistered(log types.Log) (*RegistryVoterRegistered, error) {
	event := new(RegistryVoterRegistered)
	if err := _Registry.contract.UnpackLog(event, "VoterRegistered", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// RegistryVoterRemovedIterator is returned from FilterVoterRemoved and is used to iterate over the raw logs and unpacked data for VoterRemoved events raised by the Registry contract.
type RegistryVoterRemovedIterator struct {
	Event *RegistryVoterRemoved // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *RegistryVoterRemovedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(RegistryVoterRemoved)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(RegistryVoterRemoved)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *RegistryVoterRemovedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *RegistryVoterRemovedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// RegistryVoterRemoved represents a VoterRemoved event raised by the Registry contract.
type RegistryVoterRemoved struct {
	Voter         common.Address
	RewardEpochId uint32
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterVoterRemoved is a free log retrieval operation binding the contract event 0xc68883244ac4be6ff6d4b0bf9c8f4cc0fa21af3cf2921443c1bd0af0683cfcf6.
//
// Solidity: event VoterRemoved(address indexed voter, uint32 indexed rewardEpochId)
func (_Registry *RegistryFilterer) FilterVoterRemoved(opts *bind.FilterOpts, voter []common.Address, rewardEpochId []uint32) (*RegistryVoterRemovedIterator, error) {

	var voterRule []interface{}
	for _, voterItem := range voter {
		voterRule = append(voterRule, voterItem)
	}
	var rewardEpochIdRule []interface{}
	for _, rewardEpochIdItem := range rewardEpochId {
		rewardEpochIdRule = append(rewardEpochIdRule, rewardEpochIdItem)
	}

	logs, sub, err := _Registry.contract.FilterLogs(opts, "VoterRemoved", voterRule, rewardEpochIdRule)
	if err != nil {
		return nil, err
	}
	return &RegistryVoterRemovedIterator{contract: _Registry.contract, event: "VoterRemoved", logs: logs, sub: sub}, nil
}

// WatchVoterRemoved is a free log subscription operation binding the contract event 0xc68883244ac4be6ff6d4b0bf9c8f4cc0fa21af3cf2921443c1bd0af0683cfcf6.
//
// Solidity: event VoterRemoved(address indexed voter, uint32 indexed rewardEpochId)
func (_Registry *RegistryFilterer) WatchVoterRemoved(opts *bind.WatchOpts, sink chan<- *RegistryVoterRemoved, voter []common.Address, rewardEpochId []uint32) (event.Subscription, error) {

	var voterRule []interface{}
	for _, voterItem := range voter {
		voterRule = append(voterRule, voterItem)
	}
	var rewardEpochIdRule []interface{}
	for _, rewardEpochIdItem := range rewardEpochId {
		rewardEpochIdRule = append(rewardEpochIdRule, rewardEpochIdItem)
	}

	logs, sub, err := _Registry.contract.WatchLogs(opts, "VoterRemoved", voterRule, rewardEpochIdRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(RegistryVoterRemoved)
				if err := _Registry.contract.UnpackLog(event, "VoterRemoved", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseVoterRemoved is a log parse operation binding the contract event 0xc68883244ac4be6ff6d4b0bf9c8f4cc0fa21af3cf2921443c1bd0af0683cfcf6.
//
// Solidity: event VoterRemoved(address indexed voter, uint32 indexed rewardEpochId)
func (_Registry *RegistryFilterer) ParseVoterRemoved(log types.Log) (*RegistryVoterRemoved, error) {
	event := new(RegistryVoterRemoved)
	if err := _Registry.contract.UnpackLog(event, "VoterRemoved", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
