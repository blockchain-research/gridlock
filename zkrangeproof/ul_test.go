// Copyright 2018 ING Bank N.V.
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package zkrangeproof

import (
	"crypto/rand"
	"fmt"
	"log"
	"math/big"
	"testing"
	"time"

	"github.com/blockchain-research/gridlock/crypto/bn256"
)

/*
Tests the ZK Range Proof building block, where the interval is [0, U^L).
*/
func TestZKRP_UL(t *testing.T) {
	var (
		r *big.Int
	)
	p, _ := SetupUL(10, 5)
	r, _ = rand.Int(rand.Reader, bn256.Order)
	x := new(big.Int).SetInt64(176)
	cm, _ := Commit(x, r, p.H)
	proof, _ := ProveUL(x, r, cm, p)
	proofVerifier := GenerateProofVerifier(proof)
	paramsVerifier := GenerateParamsVefifier(&p)
	result, _ := VerifyUL(&proofVerifier, paramsVerifier)
	fmt.Println("ZKRP UL result: ")
	fmt.Println(result)
	if result != true {
		t.Errorf("Assert failure: expected true, actual: %t", result)
	}
}

/*
Tests the ZK Range Proof building block, where the interval is [0, U^L).
Using marshal and unmarshal
*/
func TestZKRP_ULMarshal(t *testing.T) {
	fmt.Println(bn256.Order)
	var (
		r *big.Int
	)
	p, _ := SetupUL(10, 8)
	r, _ = rand.Int(rand.Reader, bn256.Order)
	x := new(big.Int).SetInt64(100)
	cm, _ := Commit(x, r, p.H)

	start := time.Now()
	proof, _ := ProveUL(x, r, cm, p)
	elapsed := time.Since(start)
	log.Printf("Binomial took %s", elapsed)

	proofOut := GenerateProofVerifier(proof)

	proofOutBytes := proofOut.Marshal()
	proofOut2 := new(ProofULVerifier).Unmarshal(proofOutBytes, p.l)
	fmt.Println(len(proofOutBytes))
	paramsVerifier := GenerateParamsVefifier(&p)

	paramsBytes := paramsVerifier.Marshal()
	paramsVerifier2 := new(ParamsULVerifier).Unmarshal(paramsBytes)

	start = time.Now()
	result, _ := VerifyUL(proofOut2, *paramsVerifier2)
	elapsed = time.Since(start)
	log.Printf("Binomial took %s", elapsed)

	fmt.Println("ZKRP UL result: ")
	fmt.Println(result)
	if result != true {
		t.Errorf("Assert failure: expected true, actual: %t", result)
	}
}
