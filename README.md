# Privacy-Preserving Distributed Gridlock Resolution Protocol Implementation


## Payment System Setup
In the blockchain ledger, we store each bank's account balance using pedersen commitment. It is statistically hiding and computationally binding, with additive homomorphic properties. Each payment message, tagged with a transaction id (tid: priority+timestamp), consists of the sender id, receiver id and payment amount (also in perdersen commitment). Payment messages are stored into the sender's outgoing queue, as well as receiver's incoming queue. Payments in a certain bank's ougoing queue should be settled based on FIFO, e.g., a smaller tid queued in front of a larger tid should be settled first. Gross settlement is defined as the settlement of outgoing payments of a certain bank. Gross settlement transaction contains the payload of a bank's id, payment id and a zero-knowledge range proof attesting that the post-balance after substracting the payment amount from the current balance is non-negative. Net settlement is defined as the simultaneous settlement of outgoing payments belonging to multiple banks. Gridlock is a situation where no bank can proceed to settle its outgoing queues using gross settlement, however, they can settle some of their payments simultaneously using net settlement. Gridlock resolution is an algorithm to find out the largest nettable payment set.

## Supported functionalities
`mintAccount`: central party initializes each bank's account with commitment to their balance and zkrp (balance >= 0)


`addMessage`: payer adds a payment message to the system with senderId, receiverId, commitment to payment amount and zkrp (amount >=0 )


`grossSettlement`: payer submits this transaction to settle his first payment in the outgoing queue, with zkrp (balance - amount >=0)

`startGLResolution`: coordinate starts the gridlock resolution and configures it

`proposeNettableSet`: in the distributed gridlock resolution protocol, each bank propose his own nettable outgoing set, infeasible outgoing set, with zkrp1 (balance + all incoming except in global infeasible - all nettable outgoing >= 0), with zkrp2 ( - (balance + all incoming except in global infeasible - all nettable outgoing - first payment in the infeasible outgoing queue) >= 0)

`tallyGridlockProposal`: after each round, the aggregator propose to tally the gridlock proposal of this round, the smart contract will calculate the new global infeasible set and check if it is the same as before, it converges, otherwise, it will continue to next round.

`NetGLSettlement`: when the gridlock resolution is successful, aggregator can submit a transaction to net all the gridlock resolution set, smart contract will get all proposals from ledger and settle all the payments in a single net transaction.

## Distributed Gridlock Resolution Protocol

![protocol](https://github.com/blockchain-research/gridlock/blob/master/protocol.png)

## Demo
### Mint - Add Payment - Gross Settlement
The first test case mints two accounts, adds one payment message and does gross settlement of this payment message. You can run the first demo by `go test -run TestMintAddMessageGrossSettlement`.

### Gridlock Resolution - Net Settlement
The second test case does the following things: mint five bank accounts, add ten payment messages which lead to a gridlock, distributed gridlock resolution and net settlement of the gridlock. You can run the second demo by `go test -run TestGridlockResolutionFlow`. The simulated example is as follows:


Banks | A | B | C | D | E
------------ | ------------- | ------------ | ------------- | ------------ | -------------
Account Balance| 3 | 4 | 5 | 4 | 3
T1 | -5 | +5 | | | 
T2 | | -6 | +6 | |
T3 | | -30 | +30 | |
T4 | | | -8 | +8 | 
T5 | | | -80 | | +80
T6 | | | | -7 | +7
T7 | -6 | | +6 | |
T8 | +8 | | | | -8
T9 | | +100 | | | -100
T10 | +5 | | | -5 | 

![demo](https://github.com/blockchain-research/gridlock/blob/master/demo.png)

## Libraries

### bn256

The `crypto/bn256` folder is an implementation of a particular bilinear group at the 128-bit security level. It is a modification of the official version at https://golang.org/x/crypto/bn256, which supports negative number operations. 
Note there is another implementation (https://github.com/cloudflare/bn256/blob/master/bn256.go) which claims to offer ~10 times faster performance, we might want to leverage this faster version in the future.

### zkrangeproof: Boneh-Boyen signature based
The `zkrangeproof` folder is an implementaion of zero-knowledge range proof based on Boneh-Boyen signature. It implements the paper "Efficient Protocols for Set Membership and Range Proofs" by IBM. The original implementation is from https://github.com/ing-bank/zkrangeproof/. I made a few modifications, added some functionalities (like marshaling/unmarshaling proofs) and re-factored the code a bit. The main functionalities we are using is `ul.go` and `ul_test.go`, which proves a number is within `[0,u^l)`, the proof size is `(l+2)|G2| + l|GT| + (2l+2)|BINT|`. 

Basic idea of the paper is : the veirifier fist sends the prover a Boneh-Boyen signature of every element in the set. The prover receives a signature on the particular element to which C is a commitment. The prover then “blinds” this received signature and performs a proof of knowledge that she possesses a signature on the committed element. 

### borromean ring signature based zero knowledge range proof
We have also implemented another [zero-knowledge range proof method](https://github.com/blockchain-research/crypto) described in [confidential assets](https://blockstream.com/bitcoin17-final41.pdf). This method is based on the borromean ring signature, and have more than 10 times better performance than the Boneh-Boyen signature based scheme.

### pedersen commitment
`perdersenCurve` is the pedersen commitment using the elliptic curve which aligns with the `zkrangeproof` folder, the `pedersenGroup` is another implementation of pedersen commitment based on Schnorr group. Both commitment schemes offer additive homormorphic properties and sum to zero for (x,r) and (-x,-r). Note to represent a negative integer a, we calculate a positive integer `a'` as `a'=order+a`.

### Helper commands
The command to generate protobuf go files:
`GOBIN=$GOPATH/bin PATH=$GOPATH/bin:$PATH protoc --go_out=. *.proto`
