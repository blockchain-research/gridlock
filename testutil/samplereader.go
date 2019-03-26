package testutil

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"math/big"

	"github.com/blockchain-research/gridlock/common"
	"github.com/blockchain-research/gridlock/crypto/bn256"
	"github.com/blockchain-research/gridlock/pedersencurve"
	"github.com/blockchain-research/gridlock/pedersengroup"
	pb "github.com/blockchain-research/gridlock/proto"
	"github.com/blockchain-research/gridlock/zkrangeproof"
	"github.com/golang/protobuf/proto"
)

type GLMessage struct {
	SenderId   int32
	ReceiverId int32
	Amount     *big.Int
}

type IDList struct {
	OutgoingIds   []int32
	IncomingIds   []int32
	InfeasibleIds []int32
}

var pUL, _ = zkrangeproof.SetupUL(common.U, common.L)

// SamplePedersenGroup returns a sample pedersengroup message
func SamplePedersenGroup() (_ *pb.StoredPedersenGroup, err error) {
	pedersenParams, err := pedersengroup.GeneratePedersenParams(common.BitLengthGroupOrder)
	if err != nil {
		logger.Error("Failed to Generate Pedersen params")
		return nil, err
	}

	//add pedersen public params to the ledger
	return &pb.StoredPedersenGroup{
		P: pedersenParams.Group.P.Bytes(),
		G: pedersenParams.Group.G.Bytes(),
		Q: pedersenParams.Group.Q.Bytes(),
		H: pedersenParams.H.Bytes(),
	}, nil
}

// SamplePedersen returns a sample pedersencurve message
// This setup step should be done at the client side
func SampleParamsUL() (_ []byte) {
	paramsVerifier := zkrangeproof.GenerateParamsVefifier(&pUL)
	paramsBytes := paramsVerifier.Marshal()
	return paramsBytes
}

//SampleMintAccount returns a sample MintAccount message
func SampleMintAccount(balances map[int32]*big.Int) (*pb.MintAccount, map[int32]*big.Int) {
	ma := &pb.MintAccount{
		Accounts: []*pb.BankAccount{},
	}
	randomness := map[int32]*big.Int{}
	for key, val := range balances {
		r, _ := rand.Int(rand.Reader, bn256.Order)
		randomness[key] = r
		c := pedersencurve.Commit(val, r, pUL.H)
		proof, _ := zkrangeproof.ProveUL(val, r, c, pUL)
		proofOut := zkrangeproof.GenerateProofVerifier(proof)
		ba := &pb.BankAccount{
			BankId:    key,
			CmBalance: c.Marshal(),
			Zkrp:      proofOut.Marshal(),
		}
		ma.Accounts = append(ma.Accounts, ba)
	}
	return ma, randomness
}

// SamplePaymentMessage returns a sample payment message
// The proof generation should be done at client side
func SamplePaymentMessage(paymentId int32, sender int32, receiver int32, value *big.Int) (*pb.PaymentMessage, map[int32]*big.Int) {
	r1, _ := rand.Int(rand.Reader, bn256.Order)
	c1 := pedersencurve.Commit(value, r1, pUL.H)

	proof, _ := zkrangeproof.ProveUL(value, r1, c1, pUL)
	proofOut := zkrangeproof.GenerateProofVerifier(proof)

	randomness := map[int32]*big.Int{}
	randomness[receiver] = r1
	randomness[sender] = r1
	return &pb.PaymentMessage{
		PaymentId: paymentId,
		Sender:    sender,
		Receiver:  receiver,
		CmAmount:  c1.Marshal(),
		Zkrp:      proofOut.Marshal(),
	}, randomness
}

//SampleGrossSettlementSet
func SampleGrossSettlementSet(bankId int32, payment *pb.PaymentMessage, cmBalance []byte, value *big.Int, randomness *big.Int) *pb.GrossSettlementSet {
	//calculate the sum commitment = cmBalance - outgoing cmAmount
	cmSum, _ := new(bn256.G2).Unmarshal(cmBalance)

	cmAmount, _ := new(bn256.G2).Unmarshal(payment.CmAmount)
	cmSum = new(bn256.G2).Add(cmSum, new(bn256.G2).Neg(cmAmount))

	//c := pedersencurve.Commit(value, randomness, pUL.H)
	//fmt.Println(c.Marshal())
	//fmt.Println(cmSum.Marshal())

	proof, _ := zkrangeproof.ProveUL(value, randomness, cmSum, pUL)
	proofOut := zkrangeproof.GenerateProofVerifier(proof)

	settlementSet := &pb.GrossSettlementSet{
		BankId:    bankId,
		PaymentId: payment.PaymentId,
		CmBalance: cmBalance,
		Zkrp:      proofOut.Marshal(),
	}
	return settlementSet
}

func GetStoredBankAccount(account *pb.BankAccount) []byte {
	storedBankAccount := &pb.StoredBankAccount{
		CmBalance: account.CmBalance,
	}
	storedBankAccountBytes, err := proto.Marshal(storedBankAccount)
	if err != nil {
		return nil
	}
	return storedBankAccountBytes
}

func GetStoredBankAccountFromValue(value *big.Int, r *big.Int) []byte {
	c := pedersencurve.Commit(value, r, pUL.H)
	storedBankAccount := &pb.StoredBankAccount{
		CmBalance: c.Marshal(),
	}
	storedBankAccountBytes, err := proto.Marshal(storedBankAccount)
	if err != nil {
		return nil
	}
	return storedBankAccountBytes
}

/*
* GetStoredPaymentMessage This helper function creates a pb.StoredPaymentMessage protobuf payload from a
* pb.StoredPaymentMessage protobuf payload
 */
func GetStoredPaymentMessage(paymentMessage *pb.PaymentMessage) []byte {
	storedPaymentMessage := &pb.StoredPaymentMessage{
		Sender:   paymentMessage.Sender,
		Receiver: paymentMessage.Receiver,
		CmAmount: paymentMessage.CmAmount,
		Zkrp:     paymentMessage.Zkrp,
		Status:   pb.StatusType_ACTIVE,
	}

	storedPaymentMessageBytes, err := proto.Marshal(storedPaymentMessage)
	if err != nil {
		return nil
	}
	return storedPaymentMessageBytes
}

/*
* GetStoredSettledPaymentMessage This helper function creates a pb.StoredPaymentMessage protobuf payload from a
* pb.StoredPaymentMessage protobuf payload
 */
func GetStoredSettledPaymentMessage(paymentMessage *pb.PaymentMessage) []byte {
	storedPaymentMessage := &pb.StoredPaymentMessage{
		Sender:   paymentMessage.Sender,
		Receiver: paymentMessage.Receiver,
		CmAmount: paymentMessage.CmAmount,
		Zkrp:     paymentMessage.Zkrp,
		Status:   pb.StatusType_SETTLED,
	}

	storedPaymentMessageBytes, err := proto.Marshal(storedPaymentMessage)
	if err != nil {
		return nil
	}
	return storedPaymentMessageBytes
}

func AddGridlockMessages(checker *_checker, messages map[int32]*GLMessage) (map[int32]map[int32]*big.Int, error) {
	randomnessPayment := map[int32]map[int32]*big.Int{
		1: {},
		2: {},
		3: {},
		4: {},
		5: {},
	} //bankId->paymentId->randomness
	for key, val := range messages {
		spm, randomness := SamplePaymentMessage(
			key,            //messgeId
			val.SenderId,   //payerId
			val.ReceiverId, //payeeId
			val.Amount,     //payment amount
		)
		request, err := proto.Marshal(spm)
		if err != nil {
			logger.Error("Failed to proto marshal 'PaymentMessage' object - %s", err)
			return nil, err
		}
		checker.Invoke("tx2", "addMessage",
			[]string{
				base64.StdEncoding.EncodeToString(request),
			})

		for k, v := range randomness {
			randomnessPayment[k][key] = v
		}
	}
	return randomnessPayment, nil
}

func SampleGLRConfiguration(
	gridlockId int32,
	bankIds []int32,
) *pb.GLRConfiguration {
	return &pb.GLRConfiguration{
		GridlockId: gridlockId,
		BankIds:    bankIds,
		Status:     pb.GLRStatusType_START,
	}
}

//SampleGridlockProposal returns [bankId][iteration] -> GridlockProposal for the sample testcase
func SampleGridlockProposals(
	gridlockId int32,
	balances map[int32]*big.Int,
	messages map[int32]*GLMessage,
	randomnessInit map[int32]*big.Int,
	randomnessPayment map[int32]map[int32]*big.Int,
	list map[int32]*IDList,
) (map[int32]*pb.GridlockProposal, map[int32][]byte) {
	//calculate the sum commitment = cmBalance - outgoing cmAmount
	result := map[int32]*pb.GridlockProposal{
		1: {},
		2: {},
		3: {},
		4: {},
		5: {},
	}
	postAccount := map[int32][]byte{}

	for k := range list {
		postbalance := balances[k]
		sumRandomness := randomnessInit[k]
		cmBalance := pedersencurve.Commit(balances[k], randomnessInit[k], pUL.H)
		cmSum := cmBalance

		//add incoming amounts
		for _, id := range list[k].IncomingIds {
			postbalance = new(big.Int).Add(postbalance, messages[id].Amount)
			sumRandomness = new(big.Int).Add(sumRandomness, randomnessPayment[k][id])
			amount := pedersencurve.Commit(messages[id].Amount, randomnessPayment[k][id], pUL.H)
			cmSum = new(bn256.G2).Add(cmSum, amount)
		}
		//substract outgoing amounts
		for _, id := range list[k].OutgoingIds {
			postbalance = new(big.Int).Sub(postbalance, messages[id].Amount)
			sumRandomness = new(big.Int).Sub(sumRandomness, randomnessPayment[k][id])
			amount := pedersencurve.Commit(messages[id].Amount, randomnessPayment[k][id], pUL.H)
			cmSum = new(bn256.G2).Add(cmSum, new(bn256.G2).Neg(amount))
		}
		postAccount[k] = cmSum.Marshal()
		proof, _ := zkrangeproof.ProveUL(postbalance, sumRandomness, cmSum, pUL)
		proofOut := zkrangeproof.GenerateProofVerifier(proof)

		result[k] = &pb.GridlockProposal{
			GridlockId:    gridlockId,
			BankId:        k,
			OutgoingIds:   list[k].OutgoingIds,
			InfeasibleIds: list[k].InfeasibleIds,
			CmBalance:     cmBalance.Marshal(),
			Zkrp1:         proofOut.Marshal(),
		}

		//substract the smallest id from infeasible
		if len(list[k].InfeasibleIds) > 0 {
			smallest := list[k].InfeasibleIds[0]
			for _, infeasibleId := range list[k].InfeasibleIds {
				if smallest > infeasibleId {
					smallest = infeasibleId
				}
			}
			postbalance = new(big.Int).Sub(postbalance, messages[smallest].Amount)
			postbalanceNeg := new(big.Int).Sub(new(big.Int).SetInt64(0), postbalance)
			sumRandomness = new(big.Int).Sub(sumRandomness, randomnessPayment[k][smallest])
			sumRandomnessNeg := new(big.Int).Sub(new(big.Int).SetInt64(0), sumRandomness)
			amount := pedersencurve.Commit(messages[smallest].Amount, randomnessPayment[k][smallest], pUL.H)
			cmSum = new(bn256.G2).Add(cmSum, new(bn256.G2).Neg(amount))
			cmSumNeg := new(bn256.G2).Neg(cmSum)

			proofNeg, _ := zkrangeproof.ProveUL(postbalanceNeg, sumRandomnessNeg, cmSumNeg, pUL)
			proofOutNeg := zkrangeproof.GenerateProofVerifier(proofNeg)
			result[k].Zkrp2 = proofOutNeg.Marshal()
		}
	}

	return result, postAccount
}

func CheckPostGLRAccountBalance(checker *_checker, postAccount1 map[int32][]byte, postAccount2 map[int32][]byte, postAccount3 map[int32][]byte) {
	//check the account balance are updated correctly
	storedBankAccount := &pb.StoredBankAccount{
		CmBalance: postAccount1[1],
	}
	storedBankAccountBytes, _ := proto.Marshal(storedBankAccount)
	checker.State([]byte(storedBankAccountBytes), common.AccountTable+fmt.Sprint(1))
	storedBankAccount = &pb.StoredBankAccount{
		CmBalance: postAccount2[2],
	}
	storedBankAccountBytes, _ = proto.Marshal(storedBankAccount)
	checker.State([]byte(storedBankAccountBytes), common.AccountTable+fmt.Sprint(2))
	storedBankAccount = &pb.StoredBankAccount{
		CmBalance: postAccount3[3],
	}
	storedBankAccountBytes, _ = proto.Marshal(storedBankAccount)
	checker.State([]byte(storedBankAccountBytes), common.AccountTable+fmt.Sprint(3))
	storedBankAccount = &pb.StoredBankAccount{
		CmBalance: postAccount1[4],
	}
	storedBankAccountBytes, _ = proto.Marshal(storedBankAccount)
	checker.State([]byte(storedBankAccountBytes), common.AccountTable+fmt.Sprint(4))
	storedBankAccount = &pb.StoredBankAccount{
		CmBalance: postAccount2[5],
	}
	storedBankAccountBytes, _ = proto.Marshal(storedBankAccount)
	checker.State([]byte(storedBankAccountBytes), common.AccountTable+fmt.Sprint(5))
}
