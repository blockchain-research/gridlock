package pedersencurve

import (
	"math/big"

	"github.com/blockchain-research/gridlock/crypto/bn256"
)

/*
Commit method corresponds to the Pedersen commitment scheme. Namely, given input
message x, and randomness r, it outputs g^x.h^r.
*/
func Commit(x, r *big.Int, h *bn256.G2) *bn256.G2 {
	var (
		C *bn256.G2
	)
	C = new(bn256.G2).ScalarBaseMult(x)
	C.Add(C, new(bn256.G2).ScalarMult(h, r))
	return C
}

/*
CommitG1 method corresponds to the Pedersen commitment scheme. Namely, given input
message x, and randomness r, it outputs g^x.h^r.
*/
func CommitG1(x, r *big.Int, h *bn256.G1) (*bn256.G1, error) {
	var (
		C *bn256.G1
	)
	C = new(bn256.G1).ScalarBaseMult(x)
	C.Add(C, new(bn256.G1).ScalarMult(h, r))
	return C, nil
}
