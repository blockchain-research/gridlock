package testutil

import (
	bp "bytes"
	"errors"
	"testing"

	"github.com/hyperledger/fabric/core/chaincode/shim"
)

var logger = shim.NewLogger("gridlock")

type _checker struct {
	stub *shim.MockStub
	t    *testing.T
}

func NewChecker(stub *shim.MockStub, t *testing.T) *_checker {
	return &_checker{stub: stub, t: t}
}

func (c *_checker) Init(tx string, function string, args []string) {
	var byteArgs [][]byte
	byteArgs = append(byteArgs, []byte(function))
	byteArgs = append(byteArgs, stringArrayToByteMatrix(args)...)
	response := c.stub.MockInit(tx, byteArgs)

	if response.GetStatus() != shim.OK {
		logger.Error("Init failed", errors.New(response.GetMessage()))
		c.t.FailNow()
	}
}

func stringArrayToByteMatrix(strArr []string) [][]byte {
	var byteArgs [][]byte
	for _, arg := range strArr {
		byteArgs = append(byteArgs, []byte(arg))
	}
	return byteArgs
}

func (c *_checker) Invoke(tx string, function string, args []string) {
	var byteArgs [][]byte
	byteArgs = append(byteArgs, []byte(function))
	byteArgs = append(byteArgs, stringArrayToByteMatrix(args)...)
	response := c.stub.MockInvoke(tx, byteArgs)

	if response.GetStatus() != shim.OK {
		c.t.Log("Invoke", args, "failed", errors.New(response.GetMessage()))
		c.t.FailNow()
	}
}

func (c *_checker) State(value []byte, name string) {
	bytes := c.stub.State[name]
	if bytes == nil {
		logger.Error("State", name, "failed to get value")
		c.t.FailNow()
	}
	if bp.Compare(bytes, value) != 0 {
		logger.Error("State value", bytes, "was not", value, "as expected")
		c.t.FailNow()
	}
}
