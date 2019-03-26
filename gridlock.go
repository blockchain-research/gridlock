package main

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/blockchain-research/gridlock/account"
	"github.com/blockchain-research/gridlock/common"
	"github.com/blockchain-research/gridlock/crypto/bn256"
	"github.com/blockchain-research/gridlock/message"
	"github.com/blockchain-research/gridlock/pedersengroup"
	pb "github.com/blockchain-research/gridlock/proto"
	"github.com/blockchain-research/gridlock/settlement"
	"github.com/blockchain-research/gridlock/zkrangeproof"
	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	pr "github.com/hyperledger/fabric/protos/peer"
)

var logger = shim.NewLogger("gridlock")
var pedersenParams pedersengroup.PedersenParams

// Gridlock struct
type Gridlock struct {
}

// Init function
func (t *Gridlock) Init(stub shim.ChaincodeStubInterface) pr.Response {
	logger.Info("Init Gridlock Protocol chaincode")
	return shim.Success(nil)
}

// Invoke function
func (t *Gridlock) Invoke(stub shim.ChaincodeStubInterface) pr.Response {
	function, args := stub.GetFunctionAndParameters()

	var result []byte
	var err error

	switch function {
	case "initParams":
		logger.Info("initParams")
		err = t.initParams(stub, args)
	case "mintAccount":
		logger.Info("mintAccount")
		err = account.MintAccount(stub, args)
	case "addMessage":
		logger.Info("addMessage")
		err = message.AddMessage(stub, args)
	case "grossSettlement":
		logger.Info("grossSettlement")
		err = settlement.GrossSettlement(stub, args)
	case "startGLResolution":
		logger.Info("startGLResolution")
		err = t.startGLResolution(stub, args)
	case "proposeNettableSet":
		logger.Info("proposeNettableSet")
		err = t.proposeNettableSet(stub, args)
	case "tallyGridlockProposal":
		logger.Info("tallyGridlockProposal")
		err = t.tallyGridlockProposal(stub, args)
	case "NetGLSettlement":
		logger.Info("NetGLSettlement")
		err = settlement.NetGLSettlement(stub, args)
	default:
		logger.Error(fmt.Sprintf("Invalid invocation function %s", function))
		err = fmt.Errorf("Invalid invocation function %s", function)
	}
	if err != nil {
		return shim.Error(err.Error())
	}
	return shim.Success(result)
}

func (t *Gridlock) initPedersenGroup(stub shim.ChaincodeStubInterface, args []string) error {
	if len(args) != 1 {
		return errors.New("Need exactly one argument: <base64-encoded-pedersengroupmessage-object>")
	}
	pedersenToStoreBytes, err := base64.StdEncoding.DecodeString(args[0])
	if err != nil {
		logger.Error("Failed to base64-decode protobuf-encoded pedersengroup")
		return err
	}

	err = stub.PutState(common.PedersenTable+"_GROUP", pedersenToStoreBytes)
	if err != nil {
		logger.Errorf("Failed to add to ledger")
		return err
	}
	return nil
}

func (t *Gridlock) initParams(stub shim.ChaincodeStubInterface, args []string) error {
	if len(args) != 1 {
		return errors.New("Need exactly one argument: <base64-encoded-object>")
	}
	paramsToStoreBytes, err := base64.StdEncoding.DecodeString(args[0])
	if err != nil {
		logger.Error("Failed to base64-decode protobuf-encoded pedersencurve")
		return err
	}
	err = stub.PutState(common.PedersenTable+"_CURVE", paramsToStoreBytes)
	if err != nil {
		logger.Errorf("Failed to add to ledger")
		return err
	}
	return nil
}

func (t *Gridlock) startGLResolution(stub shim.ChaincodeStubInterface, args []string) error {
	if len(args) != 1 {
		return errors.New("Need exactly one argument: <base64-encoded-object>")
	}
	configBytes, err := base64.StdEncoding.DecodeString(args[0])
	if err != nil {
		logger.Error("Failed to base64-decode protobuf-encoded glr configuration")
		return err
	}
	config := &pb.GLRConfiguration{}
	err = proto.Unmarshal(configBytes, config)
	if err != nil {
		logger.Error("Failed to unmarshal glr configuration")
		return err
	}

	//add GLR configuration to the ledger
	err = common.AddGLRConfigurationToLedger(stub, common.ConfigTable+fmt.Sprint(config.GridlockId), config)
	if err != nil {
		return err
	}

	return nil
}

func (t *Gridlock) proposeNettableSet(stub shim.ChaincodeStubInterface, args []string) error {
	if len(args) != 1 {
		return errors.New("Need exactly one argument: <base64-encoded-object>")
	}
	proposalBytes, err := base64.StdEncoding.DecodeString(args[0])
	if err != nil {
		logger.Error("Failed to base64-decode protobuf-encoded gridlockproposal")
		return err
	}
	proposal := &pb.GridlockProposal{}
	err = proto.Unmarshal(proposalBytes, proposal)
	if err != nil {
		logger.Error("Failed to unmarshal gridlockProposal")
		return err
	}

	//verify gridlock proposal
	success, err := t.verifyGridlockProposal(stub, proposal)
	if err != nil {
		return err
	}
	if success != true {
		logger.Error("Verification of gridlock proposal failed")
		return errors.New("Verification of gridlock proposal failed")
	}

	//add the gridlock proposal to the ledger
	err = common.AddGridlockProposalToLedger(
		stub,
		common.ProposalTable+fmt.Sprint(proposal.GridlockId)+fmt.Sprint(proposal.BankId),
		&pb.StoredGridlockProposal{
			OutgoingIds:   proposal.OutgoingIds,
			InfeasibleIds: proposal.InfeasibleIds,
			CmBalance:     proposal.CmBalance,
			Zkrp1:         proposal.Zkrp1,
			Zkrp2:         proposal.Zkrp2,
		})
	return nil
}

func (t *Gridlock) tallyGridlockProposal(stub shim.ChaincodeStubInterface, args []string) error {
	if len(args) != 1 {
		return errors.New("Need exactly one argument: <base64-encoded-object>")
	}
	tallyBytes, err := base64.StdEncoding.DecodeString(args[0])
	if err != nil {
		logger.Error("Failed to base64-decode protobuf-encoded tally proposal")
		return err
	}
	tally := &pb.TallyGridlockProposal{}
	err = proto.Unmarshal(tallyBytes, tally)
	if err != nil {
		logger.Error("Failed to unmarshal TallyGridlockProposal")
		return err
	}

	config, err := common.GetGLRConfigFromLedger(stub, common.ConfigTable+fmt.Sprint(tally.GridlockId))
	if config.Status != pb.GLRStatusType_START {
		logger.Error("wrong state of current glr")
		return errors.New("wrong state of current glr")
	}

	infeasibleObj, err := common.GetQueueFromLedger(stub, common.InfeasibleTable+fmt.Sprint(tally.GridlockId))
	if err != nil {
		logger.Error("Failed to read infeasible from ledger")
		return err
	}
	logger.Info(infeasibleObj)

	infeasible := []int32{}
	for _, id := range config.BankIds {
		proposal, err := common.GetGridlockProposalFromLedger(
			stub,
			common.ProposalTable+fmt.Sprint(tally.GridlockId)+fmt.Sprint(id),
		)
		if err != nil {
			return err
		}
		infeasible = append(infeasible, proposal.InfeasibleIds...)
	}
	logger.Info(infeasible)

	//add infeasible to the ledger
	err = common.AddQueueToLedger(
		stub,
		common.InfeasibleTable+fmt.Sprint(tally.GridlockId),
		&pb.StoredPaymentQueue{PaymentIds: infeasible},
	)
	if err != nil {
		return err
	}

	//check if infeasible is unchanged, mark it as SUCCESS
	if len(infeasible) == len(infeasibleObj.PaymentIds) {
		logger.Info("Converged, the gridlock resolution is successful")
		config.Status = pb.GLRStatusType_SUCCESS
		err = common.AddGLRConfigurationToLedger(stub, common.ConfigTable+fmt.Sprint(config.GridlockId), config)
		if err != nil {
			return err
		}
	}
	return nil
}

//verifyGridlockProposal verifies the zkrp1 of cmBalance-outgoing+incoming >=0
//zkrp2 of -(cmBalance-outgoing-highestInfeasible) >=0
func (t *Gridlock) verifyGridlockProposal(stub shim.ChaincodeStubInterface, proposal *pb.GridlockProposal) (bool, error) {
	//check whether the BankId is in the config table
	config, err := common.GetGLRConfigFromLedger(stub, common.ConfigTable+fmt.Sprint(proposal.GridlockId))
	if config.Status != pb.GLRStatusType_START {
		logger.Info("wrong state of current glr")
		return false, nil
	}
	isValidBankId := false
	for _, id := range config.BankIds {
		if id == proposal.BankId {
			isValidBankId = true
			break
		}
	}
	if isValidBankId == false {
		logger.Error("Invalid bankId ", proposal.BankId)
		return false, nil
	}

	//verify the priority is reserved for OutgoingIds
	success, err := settlement.VerifyStrictPriority(stub, proposal.BankId, proposal.OutgoingIds)
	if err != nil {
		return false, err
	}
	if success != true {
		logger.Error("Priority order is not reserved")
		return false, errors.New("Priority order is not reserved")
	}

	//get stored params
	paramsUL, err := common.GetParamsFromLedger(stub)
	if err != nil {
		logger.Error("Failed to read parameters from ledger")
		return false, err
	}

	//get current bank balance and check it is the same as CmBalance in the settlementSet
	account, err := common.GetAccountFromLedger(stub, common.AccountTable+fmt.Sprint(proposal.BankId))
	if err != nil {
		logger.Error("Failed to read account from ledger")
		return false, err
	}
	if bytes.Compare(account.CmBalance, proposal.CmBalance) != 0 {
		logger.Error("The cmBalance in account from ledger is different from the cmBalance in proposal")
		return false, nil
	}

	//calculate the post-balance commitment = cmBalance - outgoing cmAmount + incoming payment not in infeasible set
	infeasible, err := common.GetQueueFromLedger(stub, common.InfeasibleTable+fmt.Sprint(proposal.GridlockId))
	if err != nil {
		logger.Error("Failed to read infeasible from ledger")
		return false, err
	}

	cmSum, _ := new(bn256.G2).Unmarshal(account.CmBalance)
	//add all payments in the incoming queue excluding those in infeasible
	inQueue, err := common.GetQueueFromLedger(stub, common.InQueueTable+fmt.Sprint(proposal.BankId))
	if err != nil {
		logger.Error("Failed to read inQueue from ledger")
		return false, err
	}
	for _, id := range inQueue.PaymentIds {
		isFeasible := true
		for _, infeasibleId := range infeasible.PaymentIds {
			if id == infeasibleId {
				isFeasible = false
			}
		}
		if isFeasible == true {
			payment, err := common.GetPaymentFromLedger(stub, common.MessageTable+fmt.Sprint(id))
			if err != nil {
				return false, err
			}
			cmAmount, _ := new(bn256.G2).Unmarshal(payment.CmAmount)
			cmSum = new(bn256.G2).Add(cmSum, cmAmount)
		}
	}

	//subscract those outgoing payments from proposal.OutgoingIds
	for _, id := range proposal.OutgoingIds {
		payment, err := common.GetPaymentFromLedger(stub, common.MessageTable+fmt.Sprint(id))
		if err != nil {
			return false, err
		}
		cmAmount, _ := new(bn256.G2).Unmarshal(payment.CmAmount)
		cmSum = new(bn256.G2).Add(cmSum, new(bn256.G2).Neg(cmAmount))
	}

	//check that cmSum's range proof: zkrp1
	proof1 := new(zkrangeproof.ProofULVerifier).Unmarshal(proposal.Zkrp1, common.L)
	//verify the committed value in the proof is the same as cmBalance-outgoing cmAmount
	logger.Info("checking zkrp1")
	if bytes.Compare(proof1.C.Marshal(), cmSum.Marshal()) != 0 {
		logger.Error("The committed values does not match the one in the proof")
		return false, nil
	}
	//verify the range proof
	result, _ := zkrangeproof.VerifyUL(proof1, *paramsUL)
	if result != true {
		logger.Error("The zero knowledge range proof verification failed. The committed value in receiving amount is not within range.")
		return false, errors.New("ZKP verification failed")
	}
	logger.Info("The committed post balance after settling all outgoingIds is whtin range (0,u^l)")

	//calculate cmSum-smallestPidFromInfeasible
	if len(proposal.InfeasibleIds) == 0 {
		logger.Info("No infeasible set, no need to verify zkrp2")
		return true, nil
	}
	logger.Info("checking zkrp2")
	smallest := proposal.InfeasibleIds[0]
	for _, id := range proposal.InfeasibleIds {
		if smallest > id {
			smallest = id
		}
	}

	//subscract the amount of smallest id from cmSum
	payment, err := common.GetPaymentFromLedger(stub, common.MessageTable+fmt.Sprint(smallest))
	if err != nil {
		return false, err
	}
	cmAmount, _ := new(bn256.G2).Unmarshal(payment.CmAmount)
	cmSum = new(bn256.G2).Add(cmSum, new(bn256.G2).Neg(cmAmount))
	cmSumNeg := new(bn256.G2).Neg(cmSum)

	//check new NegcmSum's range proof: zkrp2
	proof2 := new(zkrangeproof.ProofULVerifier).Unmarshal(proposal.Zkrp2, common.L)
	//verify the committed value in the proof is the same as cmBalance-outgoing cmAmount
	if bytes.Compare(proof2.C.Marshal(), cmSumNeg.Marshal()) != 0 {
		logger.Error("The committed values does not match the one in the proof")
		return false, nil
	}
	//verify the range proof
	result, _ = zkrangeproof.VerifyUL(proof2, *paramsUL)
	if result != true {
		logger.Error("The zero knowledge range proof verification failed. The committed value is not within range.")
		return false, errors.New("ZKP verification failed")
	}
	logger.Info("The neg committed post balance after settling all outgoingIds+smallest infeasible id is whtin range (0,u^l)")

	return true, nil
}

// main function
func main() {
	err := shim.Start(new(Gridlock))
	if err != nil {
		logger.Errorf("Error starting Gridlock: %s", err)
	}
}
