package message

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/blockchain-research/gridlock/common"
	pb "github.com/blockchain-research/gridlock/proto"
	"github.com/blockchain-research/gridlock/zkrangeproof"
	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/core/chaincode/shim"
)

var logger = shim.NewLogger("gridlock")

func AddMessage(stub shim.ChaincodeStubInterface, args []string) error {
	logger.Info("add payment Message to the system")

	if len(args) != 1 {
		return errors.New("Need exactly one argument: <base64-encoded-paymentmessage-object>")
	}

	paymentMessageBytes, err := base64.StdEncoding.DecodeString(args[0])
	if err != nil {
		logger.Error("Failed to base64-decode protobuf-encoded payment message")
		return err
	}
	paymentMessage := &pb.PaymentMessage{}
	err = proto.Unmarshal(paymentMessageBytes, paymentMessage)
	if err != nil {
		logger.Error("Failed to proto-unmarshal payment message")
		return err
	}

	//verify the payment message
	success, err := verifyPaymentMessage(stub, paymentMessage)
	if err != nil {
		return err
	}
	if success != true {
		logger.Error("Verification of Payment message failed")
		return errors.New("Payment message is not valid")
	}

	//add payment message to MessageTable indexed by message id
	err = common.AddPaymentToLedger(stub,
		common.MessageTable+fmt.Sprint(paymentMessage.PaymentId),
		&pb.StoredPaymentMessage{
			Sender:   paymentMessage.Sender,
			Receiver: paymentMessage.Receiver,
			CmAmount: paymentMessage.CmAmount,
			Zkrp:     paymentMessage.Zkrp,
			Status:   pb.StatusType_ACTIVE,
		},
	)

	//add payment message to OutQueueTable indexed by Sender, FIFO model
	err = common.AddQueueElementToLedger(stub, common.OutQueueTable+fmt.Sprint(paymentMessage.Sender), paymentMessage.PaymentId)
	if err != nil {
		return err
	}

	//add payment message to InQueueTable indexed by Receiver
	err = common.AddQueueElementToLedger(stub, common.InQueueTable+fmt.Sprint(paymentMessage.Receiver), paymentMessage.PaymentId)
	if err != nil {
		return err
	}
	return nil
}

//verify payment message: sender id within range, receiver id within range, sender != receiver
//zkp committed value in cmAmount is within [0,u^l)
func verifyPaymentMessage(stub shim.ChaincodeStubInterface, paymentMessage *pb.PaymentMessage) (bool, error) {
	if paymentMessage.Sender > common.NumOfBanks || paymentMessage.Sender <= 0 {
		logger.Info("Invalid Sender %s", paymentMessage.Sender)
		return false, nil
	}
	if paymentMessage.Receiver > common.NumOfBanks || paymentMessage.Receiver <= 0 {
		logger.Info("Invalid Receiver %s", paymentMessage.Receiver)
		return false, nil
	}
	if paymentMessage.Sender == paymentMessage.Receiver {
		logger.Info("Duplicate bankId")
		return false, nil
	}

	//get stored params
	paramsUL, err := common.GetParamsFromLedger(stub)
	if err != nil {
		logger.Info("Failed to read parameters from ledger")
		return false, err
	}

	//check that cmAmount's range proof
	proof := new(zkrangeproof.ProofULVerifier).Unmarshal(paymentMessage.Zkrp, common.L)

	if bytes.Compare(proof.C.Marshal(), paymentMessage.CmAmount) != 0 {
		logger.Info("The committed values does not match the one in the proof")
		return false, nil
	}

	result, _ := zkrangeproof.VerifyUL(proof, *paramsUL)
	if result != true {
		logger.Error("The zero knowledge range proof verification failed. The committed value in receiving amount is not within range.")
		return false, errors.New("ZKP verification failed")
	}
	logger.Info("The committed receiving amount is whtin range (0,u^l)")

	return true, nil
}
