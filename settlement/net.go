package settlement

import (
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/blockchain-research/gridlock/common"
	"github.com/blockchain-research/gridlock/crypto/bn256"
	pb "github.com/blockchain-research/gridlock/proto"
	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/core/chaincode/shim"
)

func NetGLSettlement(stub shim.ChaincodeStubInterface, args []string) error {
	if len(args) != 1 {
		return errors.New("Need exactly one argument: <base64-encoded-NetGridlockProposal-object>")
	}

	netBytes, err := base64.StdEncoding.DecodeString(args[0])
	if err != nil {
		logger.Error("Failed to base64-decode protobuf-encoded NetGridlockProposal")
		return err
	}
	net := &pb.NetGridlockProposal{}
	err = proto.Unmarshal(netBytes, net)
	if err != nil {
		logger.Error("Failed to proto-unmarshal NetGridlockProposal")
		return err
	}

	//get GridlockConfiguration
	config, err := common.GetGLRConfigFromLedger(stub, common.ConfigTable+fmt.Sprint(net.GridlockId))
	if err != nil {
		return err
	}

	//check whether gridlock configuration is in right status
	if config.Status != pb.GLRStatusType_SUCCESS {
		logger.Error("GLR in wrong status")
		return err
	}

	//update bank account one by one
	bankBalance := map[int32]*bn256.G2{}
	outgoingIds := map[int32][]int32{}
	incomingIds := map[int32][]int32{}
	for _, bankId := range config.BankIds {
		//get bank account
		account, err := common.GetAccountFromLedger(stub, common.AccountTable+fmt.Sprint(bankId))
		bankBalance[bankId], _ = new(bn256.G2).Unmarshal(account.CmBalance)
		if err != nil {
			return err
		}
	}
	for _, bankId := range config.BankIds {
		//get gridlock proposal for each bank
		proposal, err := common.GetGridlockProposalFromLedger(
			stub,
			common.ProposalTable+fmt.Sprint(config.GridlockId)+fmt.Sprint(bankId),
		)
		if err != nil {
			return err
		}
		//settle all outgoingIds
		for _, pid := range proposal.OutgoingIds {
			paymentMessage, err := common.GetPaymentFromLedger(stub, common.MessageTable+fmt.Sprint(pid))
			if err != nil {
				return err
			}
			cmAmount, _ := new(bn256.G2).Unmarshal(paymentMessage.CmAmount)
			//substract amount from the sender
			bankBalance[paymentMessage.Sender] = new(bn256.G2).Add(bankBalance[paymentMessage.Sender], new(bn256.G2).Neg(cmAmount))
			//add amount to the receiver
			bankBalance[paymentMessage.Receiver] = new(bn256.G2).Add(bankBalance[paymentMessage.Receiver], cmAmount)

			//update ledger: mark PaymentMessage as settled
			err = common.MarkPaymentFromLedger(stub, common.MessageTable+fmt.Sprint(pid))
			if err != nil {
				return err
			}

			outgoingIds[paymentMessage.Sender] = append(outgoingIds[paymentMessage.Sender], pid)
			incomingIds[paymentMessage.Receiver] = append(incomingIds[paymentMessage.Receiver], pid)
		}

	}

	//update each bank'a account and queue
	for _, bankId := range config.BankIds {
		//update account
		err := common.AddAccountToLedger(
			stub,
			common.AccountTable+fmt.Sprint(bankId),
			&pb.StoredBankAccount{CmBalance: bankBalance[bankId].Marshal()},
		)
		if err != nil {
			return err
		}
		//update ledger: remove paymentId from outgoing queue in sender
		err = common.RemoveQueueElementFromLedger(stub, common.InQueueTable+fmt.Sprint(bankId), incomingIds[bankId])
		if err != nil {
			return err
		}
		//update ledger: remove paymentId from incoming queue of receiver
		err = common.RemoveQueueElementFromLedger(stub, common.OutQueueTable+fmt.Sprint(bankId), outgoingIds[bankId])
		if err != nil {
			return err
		}
	}
	return nil
}
