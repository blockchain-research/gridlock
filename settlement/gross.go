package settlement

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/blockchain-research/gridlock/common"
	"github.com/blockchain-research/gridlock/crypto/bn256"
	pb "github.com/blockchain-research/gridlock/proto"
	"github.com/blockchain-research/gridlock/zkrangeproof"
	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/core/chaincode/shim"
)

var logger = shim.NewLogger("gridlock")

func GrossSettlement(stub shim.ChaincodeStubInterface, args []string) error {
	logger.Info("GrossSettlement of a certain bank")

	if len(args) != 1 {
		return errors.New("Need exactly one argument: <base64-encoded-SettlementSet-object>")
	}

	settlementSetBytes, err := base64.StdEncoding.DecodeString(args[0])
	if err != nil {
		logger.Error("Failed to base64-decode protobuf-encoded SettlementSet")
		return err
	}
	settlementSet := &pb.GrossSettlementSet{}
	err = proto.Unmarshal(settlementSetBytes, settlementSet)
	if err != nil {
		logger.Error("Failed to proto-unmarshal settlementSet")
		return err
	}

	//Optional based on queue model: verify the strict priority
	success, err := VerifyStrictPriority(stub, settlementSet.BankId, []int32{settlementSet.PaymentId})
	if err != nil {
		return err
	}
	if success != true {
		logger.Error("Verification of strict priority failed")
		return errors.New("Verification of strict priority failed")
	}

	//verify the settlementSet
	success, err = verifySettlementSet(
		stub,
		settlementSet.BankId,
		settlementSet.CmBalance,
		settlementSet.Zkrp,
		[]int32{settlementSet.PaymentId},
	)
	if err != nil {
		return err
	}
	if success != true {
		logger.Error("Verification of settlementSet failed")
		return errors.New("Settlement Set is not valid")
	}

	//update ledger: mark PaymentMessage as settled
	err = common.MarkPaymentFromLedger(stub, common.MessageTable+fmt.Sprint(settlementSet.PaymentId))
	if err != nil {
		return err
	}

	//update ledger: update account balance of sender
	paymentMessage, err := common.GetPaymentFromLedger(stub, common.MessageTable+fmt.Sprint(settlementSet.PaymentId))
	if err != nil {
		return err
	}
	err = common.UpdateAccountFromLedger(
		stub,
		common.AccountTable+fmt.Sprint(paymentMessage.Sender),
		false, //decrease
		paymentMessage.CmAmount)
	if err != nil {
		return err
	}
	//update ledger: update account balance of receiver
	err = common.UpdateAccountFromLedger(
		stub,
		common.AccountTable+fmt.Sprint(paymentMessage.Receiver),
		true, //increase
		paymentMessage.CmAmount)
	if err != nil {
		return err
	}
	//update ledger: remove paymentId from outgoing queue in sender
	err = common.RemoveQueueElementFromLedger(stub, common.InQueueTable+fmt.Sprint(paymentMessage.Receiver), []int32{settlementSet.PaymentId})
	if err != nil {
		return err
	}

	//update ledger: remove paymentId from incoming queue of receiver
	err = common.RemoveQueueElementFromLedger(stub, common.OutQueueTable+fmt.Sprint(paymentMessage.Sender), []int32{settlementSet.PaymentId})
	if err != nil {
		return err
	}

	return nil
}

//verify settlement set: current bank balance is the same as CmBalance in settlementSet
//zkrp committed value in cmBalance-outgoing is within [0,u^l)
func verifySettlementSet(stub shim.ChaincodeStubInterface, bankId int32, cmBalance []byte, zkrp []byte, paymentIds []int32) (bool, error) {
	if bankId > common.NumOfBanks || bankId <= 0 {
		logger.Info("Invalid bankId ", bankId)
		return false, nil
	}

	//get stored params
	paramsUL, err := common.GetParamsFromLedger(stub)
	if err != nil {
		logger.Info("Failed to read parameters from ledger")
		return false, err
	}

	//get current bank balance and check it is the same as CmBalance in the settlementSet
	account, err := common.GetAccountFromLedger(stub, common.AccountTable+fmt.Sprint(bankId))
	if bytes.Compare(account.CmBalance, cmBalance) != 0 {
		logger.Info("The cmBalance in account from ledger is different from the cmBalance in settlement set")
		return false, nil
	}

	//calculate the post-balance commitment = cmBalance - outgoing cmAmount
	cmSum, _ := new(bn256.G2).Unmarshal(account.CmBalance)
	for _, id := range paymentIds {
		payment, err := common.GetPaymentFromLedger(stub, common.MessageTable+fmt.Sprint(id))
		if err != nil {
			return false, err
		}
		if payment.Sender != bankId {
			logger.Error("The paymentid in the settlementSet is not the outgoing payment of bankId", bankId)
			return false, errors.New("payment's sender is not bankId")
		}
		if payment.Status == pb.StatusType_SETTLED {
			logger.Error("The payment is already settled")
			return false, errors.New("The payment is already settled")
		}
		cmAmount, _ := new(bn256.G2).Unmarshal(payment.CmAmount)
		cmSum = new(bn256.G2).Add(cmSum, new(bn256.G2).Neg(cmAmount))
	}

	//check that cmSum's range proof
	proof := new(zkrangeproof.ProofULVerifier).Unmarshal(zkrp, common.L)
	//verify the committed value in the proof is the same as cmBalance-outgoing cmAmount
	if bytes.Compare(proof.C.Marshal(), cmSum.Marshal()) != 0 {
		logger.Info("The committed values does not match the one in the proof")
		return false, nil
	}
	//verify the range proof
	result, _ := zkrangeproof.VerifyUL(proof, *paramsUL)
	if result != true {
		logger.Error("The zero knowledge range proof verification failed. The committed value in receiving amount is not within range.")
		return false, errors.New("ZKP verification failed")
	}
	logger.Info("The committed post balance is whtin range (0,u^l)")

	return true, nil
}

//verify the settlementSet's PaymentIds are obey strict priority queue model
//i.e., if a queue has {1,2,3,4} You can only settle based on order {1,2,3} You cannot settle{1,2,4}
func VerifyStrictPriority(stub shim.ChaincodeStubInterface, bankId int32, paymentIds []int32) (bool, error) {
	outQueue, err := common.GetQueueFromLedger(stub, common.OutQueueTable+fmt.Sprint(bankId))

	if err != nil {
		logger.Error("Failed to get outgoing queue from ledger")
		return false, err
	}
	infeasible := []int32{}
	for _, qid := range outQueue.PaymentIds {
		isInfeasible := true
		for _, pid := range paymentIds {
			if qid == pid {
				isInfeasible = false
			}
		}
		if isInfeasible == true {
			infeasible = append(infeasible, qid)
		}
	}
	if len(infeasible) == 0 {
		return true, nil
	}

	//the smallest id in infeasible should be larger than the largest id in paymentIds
	smallestQid := infeasible[0]
	biggestPid := paymentIds[0]
	for _, qid := range infeasible {
		if qid < smallestQid {
			smallestQid = qid
		}
	}
	for _, pid := range paymentIds {
		if pid > biggestPid {
			biggestPid = pid
		}
	}
	if smallestQid <= biggestPid {
		return false, nil
	}

	return true, nil
}
