package pedersencurve

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"math/big"
	"testing"

	"github.com/blockchain-research/gridlock/crypto/bn256"
	"github.com/stretchr/testify/assert"
)

func TestPedersenSum(t *testing.T) {
	h := GetBigInt("18560948149108576432482904553159745978835170526553990798435819795989606410925")
	H := new(bn256.G2).ScalarBaseMult(h)
	r1, _ := rand.Int(rand.Reader, bn256.Order)
	x1 := new(big.Int).SetInt64(10)
	cm1 := Commit(x1, r1, H)

	r2, _ := rand.Int(rand.Reader, bn256.Order)
	x2 := new(big.Int).SetInt64(5)
	cm2 := Commit(x2, r2, H)

	rSum := new(big.Int).Add(r1, r2)
	xSum := new(big.Int).Add(x1, x2)
	cmSum := Commit(xSum, rSum, H)
	cmSumBytes := cmSum.Marshal()

	cm3 := new(bn256.G2).Add(cm1, cm2)
	cm3Bytes := cm3.Marshal()

	//fmt.Println("cm3=v%", cmSumBytes)
	//fmt.Println("cmsum=v%", cm3Bytes)
	success := bytes.Compare(cmSumBytes, cm3Bytes)
	assert.Equal(t, 0, success, "Pedersen commitment addition failed.")
}

func TestPedersenSub(t *testing.T) {
	h := GetBigInt("18560948149108576432482904553159745978835170526553990798435819795989606410925")
	H := new(bn256.G2).ScalarBaseMult(h)
	r1, _ := rand.Int(rand.Reader, bn256.Order)
	x1 := new(big.Int).SetInt64(10)
	cm1 := Commit(x1, r1, H)

	r2, _ := rand.Int(rand.Reader, bn256.Order)
	x2 := new(big.Int).SetInt64(5)
	cm2 := Commit(x2, r2, H)

	rSub := new(big.Int).Sub(r1, r2)
	xSub := new(big.Int).Sub(x1, x2)
	cmSub := Commit(xSub, rSub, H)
	cmSubBytes := cmSub.Marshal()

	cm3 := new(bn256.G2).Add(cm1, new(bn256.G2).Neg(cm2))
	cm3Bytes := cm3.Marshal()

	fmt.Println("cm3=v%", cmSubBytes)
	fmt.Println("cmsum=v%", cm3Bytes)
	success := bytes.Compare(cmSubBytes, cm3Bytes)
	assert.Equal(t, 0, success, "Pedersen commitment substraction failed.")
}

func TestPedersenNeg(t *testing.T) {
	h := GetBigInt("18560948149108576432482904553159745978835170526553990798435819795989606410925")
	H := new(bn256.G2).ScalarBaseMult(h)
	r1, _ := rand.Int(rand.Reader, bn256.Order)
	x1 := new(big.Int).SetInt64(10)
	cm1 := Commit(x1, r1, H)

	x2 := x1.Neg(x1)
	r2 := r1.Neg(r1)
	cm2 := Commit(x2, r2, H)

	cm3 := new(bn256.G2).Add(cm1, cm2)

	success := cm3.IsZero()
	assert.Equal(t, true, success, "Pedersen commitment zero sum failed.")
}

func GetBigInt(value string) *big.Int {
	i := new(big.Int)
	i.SetString(value, 10)
	return i
}
