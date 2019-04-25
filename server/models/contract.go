package models

import (
	"github.com/gochain-io/gochain/v3/accounts/abi"
	"time"
)

type AbiDefinition struct {
	Type      string
	Name      string
	Constant  bool
	Anonymous bool
	Inputs    []abi.Argument
	Outputs   []abi.Argument
}

type Contract struct {
	Address         string          `json:"address" bson:"address"`
	Bytecode        string          `json:"byte_code" bson:"byte_code"`
	Valid           bool            `json:"valid" bson:"valid"`
	ContractName    string          `json:"contract_name" bson:"contract_name"`
	CompilerVersion string          `json:"compiler_version" bson:"compiler_version"`
	Optimization    bool            `json:"optimization" bson:"optimization"`
	SourceCode      string          `json:"source_code" bson:"source_code"`
	Abi             []AbiDefinition `json:"abi" bson:"abi"`
	CreatedAt       time.Time       `json:"created_at" bson:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at" bson:"updated_at"`
	/*RecaptchaToken  string    `json:"recaptcha_token"`*/
}
