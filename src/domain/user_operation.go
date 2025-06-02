package domain

import (
	"encoding/json"
)

// UserOperation represents the ERC-4337 user operation structure
type UserOperation struct {
	Sender                        string      `json:"sender"`
	Nonce                         string      `json:"nonce"`
	Factory                       string      `json:"factory,omitempty"`
	FactoryData                   string      `json:"factoryData,omitempty"`
	CallData                      string      `json:"callData"`
	CallGasLimit                  string      `json:"callGasLimit"`
	VerificationGasLimit          string      `json:"verificationGasLimit"`
	PreVerificationGas            string      `json:"preVerificationGas"`
	MaxPriorityFeePerGas          string      `json:"maxPriorityFeePerGas"`
	MaxFeePerGas                  string      `json:"maxFeePerGas"`
	Paymaster                     string      `json:"paymaster,omitempty"`
	PaymasterVerificationGasLimit string      `json:"paymasterVerificationGasLimit,omitempty"`
	PaymasterPostOpGasLimit       string      `json:"paymasterPostOpGasLimit,omitempty"`
	PaymasterData                 interface{} `json:"paymasterData,omitempty"`
	Signature                     string      `json:"signature"`
}

// ToJSON serializes the user operation to JSON
func (uo *UserOperation) ToJSON() ([]byte, error) {
	return json.Marshal(uo)
}

// FromJSON deserializes JSON into the user operation
func (uo *UserOperation) FromJSON(data []byte) error {
	return json.Unmarshal(data, uo)
}
