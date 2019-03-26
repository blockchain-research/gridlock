package common

import (
	"errors"
	"math/big"

	"github.com/blockchain-research/gridlock/crypto/bn256"
	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/core/chaincode/shim"

	"github.com/blockchain-research/gridlock/pedersengroup"
	pb "github.com/blockchain-research/gridlock/proto"
	"github.com/blockchain-research/gridlock/zkrangeproof"
)

var logger = shim.NewLogger("gridlock")

//GetPedersenGroupFromLedger returns the public pedersen params
func GetPedersenGroupFromLedger(stub shim.ChaincodeStubInterface) (_ *pedersengroup.PedersenPublic, err error) {
	//get stored pedersen
	storedPedersenBytes, err := stub.GetState(PedersenTable + "_GROUP")
	if err != nil {
		logger.Error("Failed to read pedersen params")
		return nil, err
	}
	storedPedersen := &pb.StoredPedersenGroup{}
	err = proto.Unmarshal(storedPedersenBytes, storedPedersen)
	if err != nil {
		logger.Error("Failed to unmarshal stored pedersen")
		return nil, err
	}
	p := pedersengroup.GeneratePedersenFromParams(
		new(big.Int).SetBytes(storedPedersen.P),
		new(big.Int).SetBytes(storedPedersen.G),
		new(big.Int).SetBytes(storedPedersen.Q),
		new(big.Int).SetBytes(storedPedersen.H),
	)
	return p, nil
}

//GetParamsFromLedger returns the params
func GetParamsFromLedger(stub shim.ChaincodeStubInterface) (_ *zkrangeproof.ParamsULVerifier, err error) {
	//get stored pedersen
	storedBytes, err := stub.GetState(PedersenTable + "_CURVE")
	if err != nil {
		logger.Error("Failed to read pedersen curve params")
		return nil, err
	}
	paramsVerifier := new(zkrangeproof.ParamsULVerifier).Unmarshal(storedBytes)
	if err != nil {
		logger.Error("Failed to unmarshal stored params UL verifier")
		return nil, err
	}
	return paramsVerifier, nil
}

//AddAccountToLedger adds account to the ledger
func AddAccountToLedger(stub shim.ChaincodeStubInterface, key string, account *pb.StoredBankAccount) error {
	accountToStoreBytes, err := proto.Marshal(account)
	if err != nil {
		logger.Error("Failed to marshal account to store")
		return err
	}
	err = stub.PutState(key, accountToStoreBytes)
	if err != nil {
		logger.Errorf("Failed to add account to ledger")
		return err
	}
	return nil
}

//GetAccountFromLedger returns stored StoredBankAccount for key
func GetAccountFromLedger(stub shim.ChaincodeStubInterface, key string) (*pb.StoredBankAccount, error) {
	storedAccountBytes, err := stub.GetState(key)
	if err != nil {
		logger.Error("Failed to read account table")
		return nil, err
	}
	if storedAccountBytes == nil {
		logger.Error("No stored account with this key ", key)
		return nil, errors.New("No stored account with this key")
	}

	storedAccount := &pb.StoredBankAccount{}
	err = proto.Unmarshal(storedAccountBytes, storedAccount)
	if err != nil {
		logger.Error("Failed to unmarshal storedAccount")
		return nil, err
	}
	return storedAccount, nil
}

//DecreaseAccountFromLedger updates the account cmBalance
//isIncrease true, add cmAmount, isIncrease false, substract cmAmount
func UpdateAccountFromLedger(stub shim.ChaincodeStubInterface, key string, isIncrease bool, cmAmountBytes []byte) error {
	storedAccount, err := GetAccountFromLedger(stub, key)
	if err != nil {
		return err
	}
	cmBalance, _ := new(bn256.G2).Unmarshal(storedAccount.CmBalance)
	cmAmount, _ := new(bn256.G2).Unmarshal(cmAmountBytes)
	var cmPostBalance *bn256.G2
	if isIncrease == true {
		cmPostBalance = new(bn256.G2).Add(cmBalance, cmAmount)
	} else {
		cmPostBalance = new(bn256.G2).Add(cmBalance, new(bn256.G2).Neg(cmAmount))
	}

	storedAccount.CmBalance = cmPostBalance.Marshal()

	accountToStoreBytes, err := proto.Marshal(storedAccount)
	if err != nil {
		logger.Errorf("Unable to marshal stored account to protobuf")
		return err
	}
	err = stub.PutState(key, accountToStoreBytes)
	if err != nil {
		logger.Errorf("Failed to add stored account to ledger")
		return err
	}
	return nil
}

//GetPaymentFromLedger returns the stored payment for paymentId
func GetPaymentFromLedger(stub shim.ChaincodeStubInterface, key string) (*pb.StoredPaymentMessage, error) {
	storedPaymentBytes, err := stub.GetState(key)
	if err != nil {
		logger.Error("Failed to read payment table")
		return nil, err
	}
	if storedPaymentBytes == nil {
		logger.Error("No stored payment with this key ", key)
		return nil, errors.New("No stored payment with this key")
	}

	storedPayment := &pb.StoredPaymentMessage{}
	err = proto.Unmarshal(storedPaymentBytes, storedPayment)
	if err != nil {
		logger.Error("Failed to unmarshal storedPayment")
		return nil, err
	}
	return storedPayment, nil
}

//AddPaymentToLedger adds a payment message to the ledger key
func AddPaymentToLedger(stub shim.ChaincodeStubInterface, key string, paymentMessage *pb.StoredPaymentMessage) error {
	paymentMessageToStoreBytes, err := proto.Marshal(paymentMessage)
	if err != nil {
		logger.Errorf("Unable to marshal stored payment message to protobuf")
	}
	err = stub.PutState(key, paymentMessageToStoreBytes)
	if err != nil {
		logger.Errorf("Failed to add payment message to ledger")
		return err
	}
	return nil
}

//MarkPaymentFromLedger marks the stored payment status to settled
func MarkPaymentFromLedger(stub shim.ChaincodeStubInterface, key string) error {
	paymentMessage, err := GetPaymentFromLedger(stub, key)
	if err != nil {
		return err
	}
	paymentMessage.Status = pb.StatusType_SETTLED
	paymentMessageToStoreBytes, err := proto.Marshal(paymentMessage)
	if err != nil {
		logger.Errorf("Unable to marshal stored payment message to protobuf")
		return err
	}
	err = stub.PutState(key, paymentMessageToStoreBytes)
	if err != nil {
		logger.Errorf("Failed to add payment message to ledger")
		return err
	}
	return nil
}

//GetQueueFromLedger returns the stored queue for bankId
func GetQueueFromLedger(stub shim.ChaincodeStubInterface, key string) (_ *pb.StoredPaymentQueue, err error) {
	//get stored queue
	storedQueueBytes, err := stub.GetState(key)
	if err != nil {
		logger.Error("Failed to read queue table")
		return nil, err
	}
	storedQueue := &pb.StoredPaymentQueue{}
	if storedQueueBytes != nil {
		err = proto.Unmarshal(storedQueueBytes, storedQueue)
		if err != nil {
			logger.Error("Failed to unmarshal stored queue")
			return nil, err
		}
	} else {
		storedQueue.PaymentIds = []int32{}
	}
	return storedQueue, nil
}

//AddQueueToLedger adds the queue to the ledger
func AddQueueToLedger(stub shim.ChaincodeStubInterface, key string, queue *pb.StoredPaymentQueue) error {
	queueToStoreBytes, err := proto.Marshal(queue)
	if err != nil {
		logger.Errorf("Unable to marshal queue to protobuf")
	}
	err = stub.PutState(key, queueToStoreBytes)
	if err != nil {
		logger.Errorf("Failed to add queue to ledger")
		return err
	}
	return nil
}

//AddQueueElementToLedger adds a new paymentId to the queue by key
func AddQueueElementToLedger(stub shim.ChaincodeStubInterface, key string, paymentId int32) error {
	storedOutQueue, err := GetQueueFromLedger(stub, key)
	if err != nil {
		logger.Errorf("Failed to read outgoing queue from ledger")
		return err
	}

	storedOutQueue.PaymentIds = append(storedOutQueue.PaymentIds, paymentId)
	queueToStoreBytes, err := proto.Marshal(storedOutQueue)
	if err != nil {
		logger.Errorf("Unable to marshal queue to store protobuf")
		return err
	}
	err = stub.PutState(key, queueToStoreBytes)
	if err != nil {
		logger.Errorf("Failed to add out queue to ledger")
		return err
	}
	return nil
}

func RemoveQueueElementFromLedger(stub shim.ChaincodeStubInterface, key string, paymentIds []int32) error {
	storedQueueBytes, err := stub.GetState(key)
	if err != nil {
		logger.Error("Failed to read queue table")
		return err
	}
	if storedQueueBytes == nil {
		logger.Error("No stored queue with this key ", key)
		return errors.New("No stored queue with this key")
	}
	storedQueue := &pb.StoredPaymentQueue{}
	err = proto.Unmarshal(storedQueueBytes, storedQueue)
	if err != nil {
		logger.Error("Failed to unmarshal storedQueue")
		return err
	}

	newPaymentIds := []int32{}
	var toInclude bool
	for _, paymentId := range storedQueue.PaymentIds {
		toInclude = true
		for _, id := range paymentIds {
			if paymentId == id {
				toInclude = false
				break
			}
		}
		if toInclude == true {
			newPaymentIds = append(newPaymentIds, paymentId)
		}
	}
	storedQueue.PaymentIds = newPaymentIds

	queueToStoreBytes, err := proto.Marshal(storedQueue)
	if err != nil {
		logger.Errorf("Unable to marshal stored queue to protobuf")
	}
	err = stub.PutState(key, queueToStoreBytes)
	if err != nil {
		logger.Errorf("Failed to add queue to ledger")
		return err
	}

	return nil
}

//AddGLRConfigurationToLedger adds a payment message to the ledger key
func AddGLRConfigurationToLedger(stub shim.ChaincodeStubInterface, key string, config *pb.GLRConfiguration) error {
	configToStoreBytes, err := proto.Marshal(config)
	if err != nil {
		logger.Errorf("Unable to marshal stored payment message to protobuf")
	}
	err = stub.PutState(key, configToStoreBytes)
	if err != nil {
		logger.Errorf("Failed to add glr config to ledger")
		return err
	}
	return nil
}

//GetGLRConfigurationFromLedger returns the stored payment for paymentId
func GetGLRConfigFromLedger(stub shim.ChaincodeStubInterface, key string) (*pb.GLRConfiguration, error) {
	configBytes, err := stub.GetState(key)
	if err != nil {
		logger.Error("Failed to read config table")
		return nil, err
	}
	if configBytes == nil {
		logger.Error("No stored config with this key ", key)
		return nil, errors.New("No stored config with this key")
	}

	config := &pb.GLRConfiguration{}
	err = proto.Unmarshal(configBytes, config)
	if err != nil {
		logger.Error("Failed to unmarshal config")
		return nil, err
	}
	return config, nil
}

//AddGridlockProposalToLedger adds the gridlockProposal to the ledger
func AddGridlockProposalToLedger(stub shim.ChaincodeStubInterface, key string, proposal *pb.StoredGridlockProposal) error {
	proposalToStoreBytes, err := proto.Marshal(proposal)
	if err != nil {
		logger.Errorf("Unable to marshal proposal to protobuf")
	}
	err = stub.PutState(key, proposalToStoreBytes)
	if err != nil {
		logger.Errorf("Failed to add proposal to ledger")
		return err
	}
	return nil
}

//GetGridlockProposalFromLedger returns the stored gridlock proposal
func GetGridlockProposalFromLedger(stub shim.ChaincodeStubInterface, key string) (*pb.StoredGridlockProposal, error) {
	proposalBytes, err := stub.GetState(key)
	if err != nil {
		logger.Error("Failed to read proposal table")
		return nil, err
	}
	if proposalBytes == nil {
		logger.Error("No stored proposal with this key ", key)
		return nil, errors.New("No stored proposal with this key")
	}

	proposal := &pb.StoredGridlockProposal{}
	err = proto.Unmarshal(proposalBytes, proposal)
	if err != nil {
		logger.Error("Failed to unmarshal proposal")
		return nil, err
	}
	return proposal, nil
}
