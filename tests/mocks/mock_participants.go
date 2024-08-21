package mocks

import (
	"context"
	"crypto/ecdsa"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	bitvotes "local/fdc/client/attestation/bitVotes"
	"local/fdc/client/collector"
	"local/fdc/client/config"
	"local/fdc/client/timing"
	"local/fdc/contracts/fdc"
	"math/big"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/ethereum/go-ethereum/common"
)

var chainId = int64(31337)

func MockParticipants(systemConfig *config.System, participants []string, client *ethclient.Client, requestData string) {
	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		log.Fatal("Error: %s", err)
	}
	gasPrice.Mul(gasPrice, big.NewInt(2))

	fdcHub, err := fdc.NewFdc(systemConfig.Addresses.FdcContract, client)
	if err != nil {
		log.Fatal("Error: %s", err)
	}

	addresses, privateKeys, err := Participants(participants)
	if err != nil {
		log.Fatal("Error: %s", err)
	}

	first := true
	for {
		now := time.Now()

		round, err := timing.RoundIdForTimestamp(uint64(now.Unix()))
		if err != nil {
			log.Fatal("Error: %s", err)
		}

		startTime := timing.RoundStartTime(round + 1)

		timer := time.NewTimer(time.Until(time.Unix(int64(startTime+2), 0)))
		<-timer.C
		round++
		log.Info("start of round ", round)

		if !first {
			for j := range participants {
				err = sendBitvote(round-1, client, systemConfig.Addresses.SubmitContract, addresses[j], privateKeys[j], gasPrice)
				if err != nil {
					log.Error("Error: %s", err)
				} else {
					log.Infof("successfully sent bitvote for round %d by participant %d", round-1, j)
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
			log.Info("successfully submitted request in round ", round)
			break
		}
		if err != nil {
			log.Error("Error: %s", err)
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

func sendRequest(i int, client *ethclient.Client, fdcHub *fdc.Fdc, fromAddress common.Address, privateKeyECDSA *ecdsa.PrivateKey, gasPrice *big.Int, requestData string) error {
	cut := len(strconv.Itoa(i))
	data := requestData[:len(requestData)-cut] + strconv.Itoa(i)
	dataBytes, err := hex.DecodeString(data)
	if err != nil {
		return err
	}

	opts, err := bind.NewKeyedTransactorWithChainID(privateKeyECDSA, big.NewInt(chainId))
	if err != nil {
		log.Fatal("Error: %s", err)
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

func sendBitvote(round uint64, client *ethclient.Client, toAddress, fromAddress common.Address, privateKeyECDSA *ecdsa.PrivateKey, gasPrice *big.Int) error {
	bitvote := bitvotes.BitVote{Length: 1, BitVector: big.NewInt(1)}
	submit1FuncSel, err := collector.ParseFuncSel(collector.Submit1FuncSelHex)
	if err != nil {
		log.Panic("Could not parse submit1FuncSel:", err)
	}
	data := bitvote.EncodeBitVoteHex(round)
	bitvotesBytes, _ := hex.DecodeString(data)
	dataBytes := append(submit1FuncSel[:], 200)

	votingRound := make([]byte, 4)
	binary.BigEndian.PutUint32(votingRound, uint32(round)) // todo
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
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(big.NewInt(chainId)), privateKeyECDSA)
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