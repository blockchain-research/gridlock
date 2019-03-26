package common

//bank configuration related
const (
	NumOfBanks = 5
)

//crypto related
const (
	BitLengthGroupOrder = 256
)

//table names
const (
	AccountTable    = "ACCOUNT"
	MessageTable    = "PAYMENT_MESSAGE"
	InQueueTable    = "PAYMENT_QUEUE_INCOMING"
	OutQueueTable   = "PAYMENT_QUEUE_OUTGOING"
	PedersenTable   = "PEDERSEN"
	ConfigTable     = "GLR_CONFIGURATION"
	InfeasibleTable = "GLR_INFEASIBLE"
	ProposalTable   = "PROPOSAL"
)

const (
	U = 10 // range proof for (0,u^l)
	L = 10
)
