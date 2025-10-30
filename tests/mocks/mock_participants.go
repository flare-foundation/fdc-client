package mocks

import (
	"context"
	"crypto/ecdsa"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math/big"
	"strconv"
	"time"

	"github.com/flare-foundation/go-flare-common/pkg/contracts/fdchub"
	"github.com/flare-foundation/go-flare-common/pkg/logger"

	bitvotes "github.com/flare-foundation/fdc-client/client/attestation/bitVotes"
	"github.com/flare-foundation/fdc-client/client/collector"
	"github.com/flare-foundation/fdc-client/client/config"
	"github.com/flare-foundation/fdc-client/client/timing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/ethereum/go-ethereum/common"
)

var chainID = int64(31337)

func MockParticipants(systemConfig *config.System, participants []string, client *ethclient.Client, requestData string) {
	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		logger.Fatal("Error: %s", err)
	}
	gasPrice.Mul(gasPrice, big.NewInt(2))

	fdcHub, err := fdchub.NewFdcHub(systemConfig.Addresses.FdcContract, client)
	if err != nil {
		logger.Fatal("Error: %s", err)
	}

	addresses, privateKeys, err := Participants(participants)
	if err != nil {
		logger.Fatal("Error: %s", err)
	}

	first := true
	for {
		now := time.Now()

		round, err := timing.RoundIDForTS(uint64(now.Unix()))
		if err != nil {
			logger.Fatal("Error: %s", err)
		}

		startTime := timing.RoundStartTS(round + 1)

		timer := time.NewTimer(time.Until(time.Unix(int64(startTime+2), 0)))
		<-timer.C
		round++
		logger.Info("start of round ", round)

		if !first {
			for j := range participants {
				err = sendBitvote(round-1, client, systemConfig.Addresses.SubmitContract, addresses[j], privateKeys[j], gasPrice)
				if err != nil {
					logger.Error("Error: %s", err)
				} else {
					logger.Infof("successfully sent bitvote for round %d by participant %d", round-1, j)
				}
			}
		} else {
			first = false
		}

		for i := 0; i < 5; i++ {
			err = sendRequest(0, client, fdcHub, addresses[0], privateKeys[0], gasPrice, requestData)
			if err != nil {
				continue
			}
			logger.Info("successfully submitted request in round ", round)
			break
		}
		if err != nil {
			logger.Error("Error: %s", err)
			continue
		}
	}
}

func Participants(sks []string) ([]common.Address, []*ecdsa.PrivateKey, error) {
	pks := make([]common.Address, len(sks))
	privKeys := make([]*ecdsa.PrivateKey, len(sks))

	for i, privateKey := range sks {
		pks[i], privKeys[i] = PrivKeyToAddress(privateKey)
	}

	return pks, privKeys, nil
}

func sendRequest(i int, client *ethclient.Client, fdcHub *fdchub.FdcHub, fromAddress common.Address, privateKeyECDSA *ecdsa.PrivateKey, gasPrice *big.Int, requestData string) error {
	cut := len(strconv.Itoa(i))
	data := requestData[:len(requestData)-cut] + strconv.Itoa(i)
	dataBytes, err := hex.DecodeString(data)
	if err != nil {
		return err
	}

	opts, err := bind.NewKeyedTransactorWithChainID(privateKeyECDSA, big.NewInt(chainID))
	if err != nil {
		logger.Fatal("Error: %s", err)
	}
	opts.Value = big.NewInt(int64(1000000))
	opts.GasLimit = uint64(8000000)
	opts.GasPrice = gasPrice
	nonce, err := client.PendingNonceAt(context.Background(), fromAddress)
	if err != nil {
		return err
	}

	opts.Nonce = big.NewInt(int64(nonce))

	tx, err := fdcHub.RequestAttestation(opts, dataBytes)
	if err != nil {
		return err
	}

	receipt, err := bind.WaitMined(context.Background(), client, tx)
	if err != nil {
		return err
	}

	if receipt.Status == 0 {
		reason, err := GetFailingMessage(*client, tx.Hash())
		if err != nil {
			return err
		}
		return fmt.Errorf("error: Transaction fail: %s", reason)
	}

	return nil
}

func sendBitvote(round uint32, client *ethclient.Client, toAddress, fromAddress common.Address, privateKeyECDSA *ecdsa.PrivateKey, gasPrice *big.Int) error {
	bitvote := bitvotes.BitVote{Length: 1, BitVector: big.NewInt(1)}
	data := bitvote.EncodeBitVoteHex()
	bitvotesBytes, _ := hex.DecodeString(data)
	dataBytes := append(collector.Submit2FuncSel[:], 200)

	votingRound := make([]byte, 4)
	binary.BigEndian.PutUint32(votingRound, round)
	dataBytes = append(dataBytes, votingRound...)

	length := make([]byte, 2)
	binary.BigEndian.PutUint16(length, uint16(len(bitvotesBytes)))
	dataBytes = append(dataBytes, length...)

	dataBytes = append(dataBytes, bitvotesBytes...)
	// submission.NewSubmission()

	nonce, err := client.PendingNonceAt(context.Background(), fromAddress)
	if err != nil {
		return err
	}

	tx := types.NewTransaction(nonce, toAddress, big.NewInt(int64(0)), uint64(8000000), gasPrice, dataBytes)
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(big.NewInt(chainID)), privateKeyECDSA)
	if err != nil {
		return err
	}

	err = client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		return err
	}

	receipt, err := bind.WaitMined(context.Background(), client, signedTx)
	if err != nil {
		return err
	}

	if receipt.Status == 0 {
		reason, err := GetFailingMessage(*client, tx.Hash())
		if err != nil {
			return err
		}

		return fmt.Errorf("error: Transaction fail: %s", reason)
	}

	return nil
}
