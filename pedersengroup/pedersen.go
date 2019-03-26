package pedersengroup

import (
	"fmt"
	"math/big"

	"github.com/xlab-si/emmy/crypto/common"
	"github.com/xlab-si/emmy/crypto/groups"
)

// PedersenParams the parameters of pedersen commitment
type PedersenParams struct {
	Group *groups.SchnorrGroup
	H     *big.Int //H = g^a
	a     *big.Int
}

// PedersenPublic the public parameters of pedersen commitment
type PedersenPublic struct {
	Group *groups.SchnorrGroup
	H     *big.Int //H = g^a
}

// GeneratePedersenParams return new PedersenPrams struct
func GeneratePedersenParams(bitLengthGroupOrder int) (*PedersenParams, error) {
	group, err := groups.NewSchnorrGroup(bitLengthGroupOrder)
	if err != nil {
		return nil, fmt.Errorf("error when creating SchnorrGroup: %s", err)
	}
	a := common.GetRandomInt(group.Q)
	return &PedersenParams{
		Group: group,
		H:     group.Exp(group.G, a), // H = g^a
		a:     a,
	}, nil
}

// GeneratePedersenParams return new PedersenPrams struct
func GeneratePedersenFromParams(p *big.Int, g *big.Int, q *big.Int, H *big.Int) *PedersenPublic {
	group := groups.NewSchnorrGroupFromParams(p, g, q)
	return &PedersenPublic{
		Group: group,
		H:     H,
	}
}

//CalculateCommitment calculates the commitment message given value and randomness r
func (p *PedersenPublic) CalculateCommitment(val *big.Int, r *big.Int) (*big.Int, error) {
	if val.Cmp(p.Group.Q) == 1 || val.Cmp(big.NewInt(0)) == -1 {
		err := fmt.Errorf("committed value needs to be in Z_q (order of a base point)")
		return nil, err
	}

	// c = g^x * h^r
	//r := common.GetRandomInt(params.Group.Q)
	t1 := p.Group.Exp(p.Group.G, val)
	t2 := p.Group.Exp(p.H, r)
	comm := p.Group.Mul(t1, t2)
	return comm, nil
}

// SumCommitment returns the commitment to sum of two values using the additive homomorphic properties
func (p *PedersenPublic) SumCommitment(commitment1 *big.Int, commitment2 *big.Int) *big.Int {
	sumComm := p.Group.Mul(commitment1, commitment2)
	return sumComm
}

// CheckCommitment checks whether the val and r will commit to commitment
func (p *PedersenPublic) CheckCommitment(val *big.Int, r *big.Int, commitment *big.Int) bool {
	t1 := p.Group.Exp(p.Group.G, val) // g^x
	t2 := p.Group.Exp(p.H, r)         // h^r
	c := p.Group.Mul(t1, t2)          // g^x * h^r
	return c.Cmp(commitment) == 0
}
