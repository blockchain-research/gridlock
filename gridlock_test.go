package main

import (
	"encoding/base64"
	"fmt"
	"math/big"
	"testing"

	"github.com/blockchain-research/gridlock/common"
	pb "github.com/blockchain-research/gridlock/proto"
	"github.com/blockchain-research/gridlock/testutil"
	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/core/chaincode/shim"
)

func TestInit(t *testing.T) {
	target := new(Gridlock)
	stub := shim.NewMockStub("gridlock", target)
	checker := testutil.NewChecker(stub, t)
	checker.Init("tx1", "init", []string{})
}

//test mintAccount, addMessage, grossSettlement flow
func TestMintAddMessageGrossSettlement(t *testing.T) {
	target := new(Gridlock)
	stub := shim.NewMockStub("gridlock", target)
	checker := testutil.NewChecker(stub, t)

	//Get sample pedersen and call initPedersen
	p := testutil.SampleParamsUL()
	checker.Invoke("tx2", "initParams",
		[]string{
			base64.StdEncoding.EncodeToString(p),
		})

	//Get sample MintAccount
	sma, randomnessInit := testutil.SampleMintAccount(
		map[int32]*big.Int{1: new(big.Int).SetInt64(100), 2: new(big.Int).SetInt64(100)},
	)
	logger.Info(randomnessInit)

	request, err := proto.Marshal(sma)
	if err != nil {
		t.Logf("Failed to proto marshal 'MintAccount' object - %s", err)
		t.FailNow()
	}
	checker.Invoke("tx2", "mintAccount",
		[]string{
			base64.StdEncoding.EncodeToString(request),
		})
	//check accont is on ledger
	for i := range sma.Accounts {
		accountBytes := testutil.GetStoredBankAccount(sma.Accounts[i])
		checker.State([]byte(accountBytes), common.AccountTable+fmt.Sprint(sma.Accounts[i].BankId))
		logger.Info("Account is on the ledger", i)
	}

	//Get sample payment message and invoke addMessage
	spm, randomnessPayment := testutil.SamplePaymentMessage(
		1, //messgeId
		1, //payerId
		2, //payeeId
		new(big.Int).SetInt64(10), //payment amount
	)
	logger.Info(randomnessPayment)
	request, err = proto.Marshal(spm)
	if err != nil {
		t.Logf("Failed to proto marshal 'PaymentMessage' object - %s", err)
		t.FailNow()
	}
	checker.Invoke("tx2", "addMessage",
		[]string{
			base64.StdEncoding.EncodeToString(request),
		})

	//check payment message is stored correctly
	paymentMessageBytes := testutil.GetStoredPaymentMessage(spm)
	checker.State([]byte(paymentMessageBytes), common.MessageTable+fmt.Sprint(spm.PaymentId))
	logger.Info("Message is on the ledger")

	queue := &pb.StoredPaymentQueue{
		PaymentIds: []int32{spm.PaymentId},
	}
	storedQueueBytes, err := proto.Marshal(queue)
	if err != nil {
		t.Logf("Failed to proto marshal 'StoredPaymentQueue' object - %s", err)
		t.FailNow()
	}
	//check outgoing queue is stored correctly
	checker.State([]byte(storedQueueBytes), common.OutQueueTable+fmt.Sprint(spm.Sender))
	logger.Info("Outqueue is on the ledger")
	//check incoming queue is stored correctly
	checker.State([]byte(storedQueueBytes), common.InQueueTable+fmt.Sprint(spm.Receiver))
	logger.Info("Inqueue is on the ledger")

	//Get sample grosssettlement set
	sss := testutil.SampleGrossSettlementSet(
		1,   //bankId
		spm, //payments
		sma.Accounts[0].CmBalance, //current account balance cm
		new(big.Int).SetInt64(90),
		new(big.Int).Sub(randomnessInit[1], randomnessPayment[1]), //randomness in account + message
	)
	request, err = proto.Marshal(sss)
	if err != nil {
		t.Logf("Failed to proto marshal 'GrossSettlementSet' object - %s", err)
		t.FailNow()
	}
	checker.Invoke("tx2", "grossSettlement",
		[]string{
			base64.StdEncoding.EncodeToString(request),
		})

	//check payment message status is updated
	paymentMessageBytes = testutil.GetStoredSettledPaymentMessage(spm)
	checker.State([]byte(paymentMessageBytes), common.MessageTable+fmt.Sprint(spm.PaymentId))
	logger.Info("Message is marked settled on the ledger")

	//check sender account balance is updated
	accountBytes := testutil.GetStoredBankAccountFromValue(
		new(big.Int).SetInt64(90),                                 //value in account + message
		new(big.Int).Sub(randomnessInit[1], randomnessPayment[1]), //randomness in account + message
	)
	checker.State([]byte(accountBytes), common.AccountTable+fmt.Sprint(spm.Sender))
	logger.Info("Account of sender is updated correctly on the ledger")

	//check receiver account balance is updated
	accountBytes = testutil.GetStoredBankAccountFromValue(
		new(big.Int).SetInt64(110),                                //value in account + message
		new(big.Int).Add(randomnessInit[2], randomnessPayment[2]), //randomness in account + message
	)
	checker.State([]byte(accountBytes), common.AccountTable+fmt.Sprint(spm.Receiver))
	logger.Info("Account of receiver is updated correctly on the ledger")

	queue = &pb.StoredPaymentQueue{
		PaymentIds: []int32{},
	}
	storedQueueBytes, err = proto.Marshal(queue)
	if err != nil {
		t.Logf("Failed to proto marshal 'StoredPaymentQueue' object - %s", err)
		t.FailNow()
	}
	//check the outgoing queue of sender is updated
	checker.State([]byte(storedQueueBytes), common.OutQueueTable+fmt.Sprint(spm.Sender))
	logger.Info("Outqueue is updated on the ledger")
	//check incoming queue of receiver is updated
	checker.State([]byte(storedQueueBytes), common.InQueueTable+fmt.Sprint(spm.Receiver))
	logger.Info("Inqueue is updated on the ledger")
}

//test basic gridlock resolution and netSettlement flow
//Banks				1001	1002	1003	1004	1005
//AccountBalance	3		4		5		4		3
//T1				-5		+5
//T2						-6		+6
//T3						-30		+30
//T4								-8		+8
//T5								-80				+80
//T6										-7		+7
//T7				-6				+6
//T8				+8								-8
//T9						+100					-100
//T10				+5						-5
func TestGridlockResolutionFlow(t *testing.T) {
	var glrId int32
	glrId = 1001
	bankIds := []int32{1, 2, 3, 4, 5}
	balances := map[int32]*big.Int{
		1: new(big.Int).SetInt64(3),
		2: new(big.Int).SetInt64(4),
		3: new(big.Int).SetInt64(5),
		4: new(big.Int).SetInt64(4),
		5: new(big.Int).SetInt64(3),
	}
	messages := map[int32]*testutil.GLMessage{
		1:  &testutil.GLMessage{SenderId: 1, ReceiverId: 2, Amount: new(big.Int).SetInt64(5)},
		2:  &testutil.GLMessage{SenderId: 2, ReceiverId: 3, Amount: new(big.Int).SetInt64(6)},
		3:  &testutil.GLMessage{SenderId: 2, ReceiverId: 3, Amount: new(big.Int).SetInt64(30)},
		4:  &testutil.GLMessage{SenderId: 3, ReceiverId: 4, Amount: new(big.Int).SetInt64(8)},
		5:  &testutil.GLMessage{SenderId: 3, ReceiverId: 5, Amount: new(big.Int).SetInt64(80)},
		6:  &testutil.GLMessage{SenderId: 4, ReceiverId: 5, Amount: new(big.Int).SetInt64(7)},
		7:  &testutil.GLMessage{SenderId: 1, ReceiverId: 3, Amount: new(big.Int).SetInt64(6)},
		8:  &testutil.GLMessage{SenderId: 5, ReceiverId: 1, Amount: new(big.Int).SetInt64(8)},
		9:  &testutil.GLMessage{SenderId: 5, ReceiverId: 2, Amount: new(big.Int).SetInt64(100)},
		10: &testutil.GLMessage{SenderId: 4, ReceiverId: 1, Amount: new(big.Int).SetInt64(5)},
	}

	target := new(Gridlock)
	stub := shim.NewMockStub("gridlock", target)
	checker := testutil.NewChecker(stub, t)

	//Get sample pedersen and call initPedersen
	p := testutil.SampleParamsUL()
	checker.Invoke("tx2", "initParams",
		[]string{
			base64.StdEncoding.EncodeToString(p),
		})

	//Get sample MintAccount
	sma, randomnessInit := testutil.SampleMintAccount(balances)
	request, err := proto.Marshal(sma)
	if err != nil {
		t.Logf("Failed to proto marshal 'MintAccount' object - %s", err)
		t.FailNow()
	}
	checker.Invoke("tx2", "mintAccount",
		[]string{
			base64.StdEncoding.EncodeToString(request),
		})

	//Get sample payment message and invoke addMessage
	randomnessPayment, err := testutil.AddGridlockMessages(checker, messages)
	if err != nil {
		t.Logf("Failed to simulate gridlock - %s", err)
		t.FailNow()
	}

	//start gridlock resolution: startGR
	sc := testutil.SampleGLRConfiguration(glrId, bankIds)
	request, err = proto.Marshal(sc)
	if err != nil {
		t.Logf("Failed to proto marshal 'GLRConfiguration' object - %s", err)
		t.FailNow()
	}
	checker.Invoke("tx2", "startGLResolution",
		[]string{
			base64.StdEncoding.EncodeToString(request),
		})

	//Round1 proposal: proposeNettableSet
	list1 := map[int32]*testutil.IDList{
		1: &testutil.IDList{OutgoingIds: []int32{1, 7}, IncomingIds: []int32{8, 10}, InfeasibleIds: []int32{}},
		2: &testutil.IDList{OutgoingIds: []int32{2, 3}, IncomingIds: []int32{1, 9}, InfeasibleIds: []int32{}},
		3: &testutil.IDList{OutgoingIds: []int32{4}, IncomingIds: []int32{2, 3, 7}, InfeasibleIds: []int32{5}},
		4: &testutil.IDList{OutgoingIds: []int32{6, 10}, IncomingIds: []int32{4}, InfeasibleIds: []int32{}},
		5: &testutil.IDList{OutgoingIds: []int32{8}, IncomingIds: []int32{5, 6}, InfeasibleIds: []int32{9}},
	}

	sgp, postAccount1 := testutil.SampleGridlockProposals(glrId, balances, messages, randomnessInit, randomnessPayment, list1)
	for k := range list1 {
		request, err = proto.Marshal(sgp[k])
		if err != nil {
			t.Logf("Failed to proto marshal 'GridlockProposal' object - %s", err)
			t.FailNow()
		}
		checker.Invoke("tx2", "proposeNettableSet",
			[]string{
				base64.StdEncoding.EncodeToString(request),
			})
	}

	//Round1 tally: tallyNettableSet
	tally := &pb.TallyGridlockProposal{GridlockId: glrId}
	request, err = proto.Marshal(tally)
	if err != nil {
		t.Logf("Failed to proto marshal 'TallyGridlockProposal' object - %s", err)
		t.FailNow()
	}
	checker.Invoke("tx2", "tallyGridlockProposal",
		[]string{
			base64.StdEncoding.EncodeToString(request),
		})

	//Round2 proposal: proposeNettableSet
	list2 := map[int32]*testutil.IDList{
		2: &testutil.IDList{OutgoingIds: []int32{2}, IncomingIds: []int32{1}, InfeasibleIds: []int32{3}},
		5: &testutil.IDList{OutgoingIds: []int32{8}, IncomingIds: []int32{6}, InfeasibleIds: []int32{9}},
	}
	sgp, postAccount2 := testutil.SampleGridlockProposals(glrId, balances, messages, randomnessInit, randomnessPayment, list2)
	for k := range list2 {
		request, err = proto.Marshal(sgp[k])
		if err != nil {
			t.Logf("Failed to proto marshal 'GridlockProposal' object - %s", err)
			t.FailNow()
		}
		checker.Invoke("tx2", "proposeNettableSet",
			[]string{
				base64.StdEncoding.EncodeToString(request),
			})
	}

	//Round2 tally: tallyNettableSet
	request, err = proto.Marshal(tally)
	if err != nil {
		t.Logf("Failed to proto marshal 'TallyGridlockProposal' object - %s", err)
		t.FailNow()
	}
	checker.Invoke("tx2", "tallyGridlockProposal",
		[]string{
			base64.StdEncoding.EncodeToString(request),
		})

	//Round3 proposal: proposeNettableSet
	list3 := map[int32]*testutil.IDList{
		3: &testutil.IDList{OutgoingIds: []int32{4}, IncomingIds: []int32{2, 7}, InfeasibleIds: []int32{5}},
	}
	sgp, postAccount3 := testutil.SampleGridlockProposals(glrId, balances, messages, randomnessInit, randomnessPayment, list3)
	for k := range list3 {
		request, err = proto.Marshal(sgp[k])
		if err != nil {
			t.Logf("Failed to proto marshal 'GridlockProposal' object - %s", err)
			t.FailNow()
		}
		checker.Invoke("tx2", "proposeNettableSet",
			[]string{
				base64.StdEncoding.EncodeToString(request),
			})
	}

	//Round3 tally: tallyNettableSet
	request, err = proto.Marshal(tally)
	if err != nil {
		t.Logf("Failed to proto marshal 'TallyGridlockProposal' object - %s", err)
		t.FailNow()
	}
	checker.Invoke("tx2", "tallyGridlockProposal",
		[]string{
			base64.StdEncoding.EncodeToString(request),
		})

	//NetSettlement
	net := &pb.NetGridlockProposal{
		GridlockId: glrId,
	}
	request, err = proto.Marshal(net)
	if err != nil {
		t.Logf("Failed to proto marshal 'NetGLSettlement' object - %s", err)
		t.FailNow()
	}
	checker.Invoke("tx2", "NetGLSettlement",
		[]string{
			base64.StdEncoding.EncodeToString(request),
		})
	testutil.CheckPostGLRAccountBalance(checker, postAccount1, postAccount2, postAccount3)
}
