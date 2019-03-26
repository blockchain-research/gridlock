package pedersengroup

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/xlab-si/emmy/crypto/common"
)

func TestPedersenSum(t *testing.T) {
	bitLengthGroupOrder := 256
	pp, err := GeneratePedersenParams(bitLengthGroupOrder)
	if err != nil {
		t.Errorf("Error in GeneratePedersenParams: %v", err)
	}
	//get public parameters
	p := GeneratePedersenFromParams(pp.Group.P, pp.Group.G, pp.Group.Q, pp.H)

	a1 := common.GetRandomInt(p.Group.Q)
	r1 := common.GetRandomInt(p.Group.Q)
	c1, err := p.CalculateCommitment(a1, r1)
	if err != nil {
		t.Errorf("Error in CalculateCommitment: %v", err)
	}

	a2 := common.GetRandomInt(p.Group.Q)
	r2 := common.GetRandomInt(p.Group.Q)
	c2, err := p.CalculateCommitment(a2, r2)
	if err != nil {
		t.Errorf("Error in CalculateCommitment: %v", err)
	}

	committerSum := p.SumCommitment(c1, c2)

	success := p.CheckCommitment(p.Group.Add(a1, a2), p.Group.Add(r1, r2), committerSum)
	assert.Equal(t, true, success, "Pedersen commitment failed.")
}

func TestPedersenSumZero(t *testing.T) {
	bitLengthGroupOrder := 256
	pp, err := GeneratePedersenParams(bitLengthGroupOrder)
	if err != nil {
		t.Errorf("Error in GeneratePedersenParams: %v", err)
	}

	//get public parameters
	p := GeneratePedersenFromParams(pp.Group.P, pp.Group.G, pp.Group.Q, pp.H)

	a1 := common.GetRandomInt(p.Group.Q)
	fmt.Println(a1)
	r1 := common.GetRandomInt(p.Group.Q)
	c1, err := p.CalculateCommitment(a1, r1)
	if err != nil {
		t.Errorf("Error in CalculateCommitment: %v", err)
	}

	a2 := a1.Neg(a1)
	r2 := r1.Neg(r1)
	a2 = a2.Mod(a2, p.Group.Q)
	r2 = r2.Mod(r2, p.Group.Q)
	c2, err := p.CalculateCommitment(a2, r2)
	if err != nil {
		t.Errorf("Error in CalculateCommitment: %v", err)
	}

	committmentSum := p.SumCommitment(c1, c2)
	fmt.Println(committmentSum)
	x := new(big.Int).SetInt64(1)
	success := committmentSum.Cmp(x) == 0
	assert.Equal(t, true, success, "Pedersen commitment failed.")
}
