package account

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

func MintAccount(stub shim.ChaincodeStubInterface, args []string) error {
	logger.Info("Mint accounts")
	if len(args) != 1 {
		return errors.New("Need exactly one argument: <base64-encoded-object>")
	}
	mintAccountBytes, err := base64.StdEncoding.DecodeString(args[0])
	if err != nil {
		logger.Error("Failed to base64-decode protobuf-encoded MintAccount")
		return err
	}
	mintAccount := &pb.MintAccount{}
	err = proto.Unmarshal(mintAccountBytes, mintAccount)
	if err != nil {
		logger.Error("Failed to unmarshal MintAccount")
	}
	for _, account := range mintAccount.Accounts {
		//verify the account
		success, err := verifyAccount(stub, account)
		if err != nil {
			return err
		}
		if success != true {
			logger.Error("Verification of account failed")
			return errors.New("MintAcount is not valid")
		}
		//update account
		err = common.AddAccountToLedger(
			stub,
			common.AccountTable+fmt.Sprint(account.BankId),
			&pb.StoredBankAccount{
				CmBalance: account.CmBalance,
			},
		)
	}
	return nil
}

func verifyAccount(stub shim.ChaincodeStubInterface, account *pb.BankAccount) (bool, error) {

	if account.BankId > common.NumOfBanks || account.BankId <= 0 {
		logger.Info("Invalid bank Id %s", account.BankId)
		return false, nil
	}

	//get stored params
	paramsUL, err := common.GetParamsFromLedger(stub)
	if err != nil {
		logger.Info("Failed to read parameters from ledger")
		return false, err
	}

	//check cmBalance's range proof
	proof := new(zkrangeproof.ProofULVerifier).Unmarshal(account.Zkrp, common.L)

	if bytes.Compare(proof.C.Marshal(), account.CmBalance) != 0 {
		logger.Info("The committed values does not match the one in the proof")
		return false, nil
	}

	result, _ := zkrangeproof.VerifyUL(proof, *paramsUL)
	if result != true {
		logger.Error("The zero knowledge range proof verification failed. The committed value in account balance is not within range.")
		return false, errors.New("ZKP verification failed")
	}
	logger.Info("The committed account balance is whtin range (0,u^l)")

	return true, nil
}
