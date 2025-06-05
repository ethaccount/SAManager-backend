package erc4337

import (
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
)

// EntryPointV07 address constant
var EntryPointV07 = common.HexToAddress("0x0000000071727De22E5E9d8BAf0edAc6f37da032")

// UserOperation represents the ERC-4337 user operation structure
type UserOperation struct {
	Sender                        common.Address  `json:"sender"`
	Nonce                         *hexutil.Big    `json:"nonce"`
	Factory                       *common.Address `json:"factory"`
	FactoryData                   hexutil.Bytes   `json:"factoryData"`
	CallData                      hexutil.Bytes   `json:"callData"`
	CallGasLimit                  *hexutil.Big    `json:"callGasLimit"`
	VerificationGasLimit          *hexutil.Big    `json:"verificationGasLimit"`
	PreVerificationGas            *hexutil.Big    `json:"preVerificationGas"`
	MaxPriorityFeePerGas          *hexutil.Big    `json:"maxPriorityFeePerGas"`
	MaxFeePerGas                  *hexutil.Big    `json:"maxFeePerGas"`
	Paymaster                     *common.Address `json:"paymaster"`
	PaymasterVerificationGasLimit *hexutil.Big    `json:"paymasterVerificationGasLimit"`
	PaymasterPostOpGasLimit       *hexutil.Big    `json:"paymasterPostOpGasLimit"`
	PaymasterData                 hexutil.Bytes   `json:"paymasterData"`
	Signature                     hexutil.Bytes   `json:"signature"`
}

// MarshalJSON implements custom JSON marshaling for UserOperation
func (uo *UserOperation) MarshalJSON() ([]byte, error) {
	type Alias UserOperation
	aux := struct {
		Nonce                         string `json:"nonce"`
		CallGasLimit                  string `json:"callGasLimit"`
		VerificationGasLimit          string `json:"verificationGasLimit"`
		PreVerificationGas            string `json:"preVerificationGas"`
		MaxPriorityFeePerGas          string `json:"maxPriorityFeePerGas"`
		MaxFeePerGas                  string `json:"maxFeePerGas"`
		PaymasterVerificationGasLimit string `json:"paymasterVerificationGasLimit"`
		PaymasterPostOpGasLimit       string `json:"paymasterPostOpGasLimit"`
		*Alias
	}{
		Alias: (*Alias)(uo),
	}

	// Handle nonce with 32-byte padding
	if uo.Nonce != nil {
		// Convert to bytes and pad to 32 bytes
		nonceBytes := (*big.Int)(uo.Nonce).Bytes()
		paddedNonce := make([]byte, 32)
		copy(paddedNonce[32-len(nonceBytes):], nonceBytes)
		aux.Nonce = fmt.Sprintf("0x%064x", new(big.Int).SetBytes(paddedNonce))
	} else {
		// Handle nil nonce as zero with padding
		aux.Nonce = "0x0000000000000000000000000000000000000000000000000000000000000000"
	}

	// Handle other numeric fields without padding
	if uo.CallGasLimit != nil {
		aux.CallGasLimit = fmt.Sprintf("0x%x", (*big.Int)(uo.CallGasLimit))
	}
	if uo.VerificationGasLimit != nil {
		aux.VerificationGasLimit = fmt.Sprintf("0x%x", (*big.Int)(uo.VerificationGasLimit))
	}
	if uo.PreVerificationGas != nil {
		aux.PreVerificationGas = fmt.Sprintf("0x%x", (*big.Int)(uo.PreVerificationGas))
	}
	if uo.MaxPriorityFeePerGas != nil {
		aux.MaxPriorityFeePerGas = fmt.Sprintf("0x%x", (*big.Int)(uo.MaxPriorityFeePerGas))
	}
	if uo.MaxFeePerGas != nil {
		aux.MaxFeePerGas = fmt.Sprintf("0x%x", (*big.Int)(uo.MaxFeePerGas))
	}
	if uo.PaymasterVerificationGasLimit != nil {
		aux.PaymasterVerificationGasLimit = fmt.Sprintf("0x%x", (*big.Int)(uo.PaymasterVerificationGasLimit))
	}
	if uo.PaymasterPostOpGasLimit != nil {
		aux.PaymasterPostOpGasLimit = fmt.Sprintf("0x%x", (*big.Int)(uo.PaymasterPostOpGasLimit))
	}

	return json.Marshal(aux)
}

// UnmarshalJSON implements custom JSON unmarshaling for UserOperation
func (uo *UserOperation) UnmarshalJSON(data []byte) error {
	type Alias UserOperation
	aux := struct {
		Nonce                         string `json:"nonce"`
		CallGasLimit                  string `json:"callGasLimit"`
		VerificationGasLimit          string `json:"verificationGasLimit"`
		PreVerificationGas            string `json:"preVerificationGas"`
		MaxPriorityFeePerGas          string `json:"maxPriorityFeePerGas"`
		MaxFeePerGas                  string `json:"maxFeePerGas"`
		PaymasterVerificationGasLimit string `json:"paymasterVerificationGasLimit"`
		PaymasterPostOpGasLimit       string `json:"paymasterPostOpGasLimit"`
		*Alias
	}{
		Alias: (*Alias)(uo),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// Helper function to parse hex string to big.Int
	parseHexBig := func(hexStr string) (*big.Int, error) {
		if hexStr == "" {
			return big.NewInt(0), nil
		}
		// Remove 0x prefix if present
		if len(hexStr) >= 2 && hexStr[:2] == "0x" {
			hexStr = hexStr[2:]
		}
		// Handle empty string after removing 0x
		if hexStr == "" {
			return big.NewInt(0), nil
		}
		// Parse using big.Int SetString with base 16
		result := new(big.Int)
		_, ok := result.SetString(hexStr, 16)
		if !ok {
			return nil, fmt.Errorf("invalid hex string: %s", hexStr)
		}
		return result, nil
	}

	// Parse numeric fields
	if aux.Nonce != "" {
		nonce, err := parseHexBig(aux.Nonce)
		if err != nil {
			return fmt.Errorf("invalid nonce: %v", err)
		}
		uo.Nonce = (*hexutil.Big)(nonce)
	}

	if aux.CallGasLimit != "" {
		callGasLimit, err := parseHexBig(aux.CallGasLimit)
		if err != nil {
			return fmt.Errorf("invalid callGasLimit: %v", err)
		}
		uo.CallGasLimit = (*hexutil.Big)(callGasLimit)
	}

	if aux.VerificationGasLimit != "" {
		verificationGasLimit, err := parseHexBig(aux.VerificationGasLimit)
		if err != nil {
			return fmt.Errorf("invalid verificationGasLimit: %v", err)
		}
		uo.VerificationGasLimit = (*hexutil.Big)(verificationGasLimit)
	}

	if aux.PreVerificationGas != "" {
		preVerificationGas, err := parseHexBig(aux.PreVerificationGas)
		if err != nil {
			return fmt.Errorf("invalid preVerificationGas: %v", err)
		}
		uo.PreVerificationGas = (*hexutil.Big)(preVerificationGas)
	}

	if aux.MaxPriorityFeePerGas != "" {
		maxPriorityFeePerGas, err := parseHexBig(aux.MaxPriorityFeePerGas)
		if err != nil {
			return fmt.Errorf("invalid maxPriorityFeePerGas: %v", err)
		}
		uo.MaxPriorityFeePerGas = (*hexutil.Big)(maxPriorityFeePerGas)
	}

	if aux.MaxFeePerGas != "" {
		maxFeePerGas, err := parseHexBig(aux.MaxFeePerGas)
		if err != nil {
			return fmt.Errorf("invalid maxFeePerGas: %v", err)
		}
		uo.MaxFeePerGas = (*hexutil.Big)(maxFeePerGas)
	}

	if aux.PaymasterVerificationGasLimit != "" {
		paymasterVerificationGasLimit, err := parseHexBig(aux.PaymasterVerificationGasLimit)
		if err != nil {
			return fmt.Errorf("invalid paymasterVerificationGasLimit: %v", err)
		}
		uo.PaymasterVerificationGasLimit = (*hexutil.Big)(paymasterVerificationGasLimit)
	}

	if aux.PaymasterPostOpGasLimit != "" {
		paymasterPostOpGasLimit, err := parseHexBig(aux.PaymasterPostOpGasLimit)
		if err != nil {
			return fmt.Errorf("invalid paymasterPostOpGasLimit: %v", err)
		}
		uo.PaymasterPostOpGasLimit = (*hexutil.Big)(paymasterPostOpGasLimit)
	}

	return nil
}

// PackedUserOp represents the packed version of UserOperation for ERC-4337
type PackedUserOp struct {
	Sender             common.Address `json:"sender"`
	Nonce              *big.Int       `json:"nonce"`
	InitCode           hexutil.Bytes  `json:"initCode"`
	CallData           hexutil.Bytes  `json:"callData"`
	AccountGasLimits   hexutil.Bytes  `json:"accountGasLimits"`
	PreVerificationGas *big.Int       `json:"preVerificationGas"`
	GasFees            hexutil.Bytes  `json:"gasFees"`
	PaymasterAndData   hexutil.Bytes  `json:"paymasterAndData"`
	Signature          hexutil.Bytes  `json:"signature"`
}

// MarshalJSON implements custom JSON marshaling for PackedUserOp
func (puo *PackedUserOp) MarshalJSON() ([]byte, error) {
	type Alias PackedUserOp
	aux := struct {
		Nonce              string `json:"nonce"`
		PreVerificationGas string `json:"preVerificationGas"`
		*Alias
	}{
		Alias: (*Alias)(puo),
	}

	// Convert Nonce to hex string
	if puo.Nonce != nil {
		aux.Nonce = fmt.Sprintf("0x%x", puo.Nonce)
	} else {
		aux.Nonce = "0x0"
	}

	// Convert PreVerificationGas to hex string
	if puo.PreVerificationGas != nil {
		aux.PreVerificationGas = fmt.Sprintf("0x%x", puo.PreVerificationGas)
	} else {
		aux.PreVerificationGas = "0x0"
	}

	return json.Marshal(aux)
}

// UnmarshalJSON implements custom JSON unmarshaling for PackedUserOp
func (puo *PackedUserOp) UnmarshalJSON(data []byte) error {
	type Alias PackedUserOp
	aux := struct {
		Nonce              string `json:"nonce"`
		PreVerificationGas string `json:"preVerificationGas"`
		*Alias
	}{
		Alias: (*Alias)(puo),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// Helper function to parse hex string to big.Int
	parseHexBig := func(hexStr string) (*big.Int, error) {
		if hexStr == "" {
			return big.NewInt(0), nil
		}
		// Remove 0x prefix if present
		if len(hexStr) >= 2 && hexStr[:2] == "0x" {
			hexStr = hexStr[2:]
		}
		// Handle empty string after removing 0x
		if hexStr == "" {
			return big.NewInt(0), nil
		}
		// Parse using big.Int SetString with base 16
		result := new(big.Int)
		_, ok := result.SetString(hexStr, 16)
		if !ok {
			return nil, fmt.Errorf("invalid hex string: %s", hexStr)
		}
		return result, nil
	}

	// Parse Nonce
	if aux.Nonce != "" {
		nonce, err := parseHexBig(aux.Nonce)
		if err != nil {
			return fmt.Errorf("invalid nonce: %v", err)
		}
		puo.Nonce = nonce
	}

	// Parse PreVerificationGas
	if aux.PreVerificationGas != "" {
		preVerificationGas, err := parseHexBig(aux.PreVerificationGas)
		if err != nil {
			return fmt.Errorf("invalid preVerificationGas: %v", err)
		}
		puo.PreVerificationGas = preVerificationGas
	}

	return nil
}

// PackUserOp packs a UserOperation into a PackedUserOp according to ERC-4337 specification
func (uo *UserOperation) PackUserOp() *PackedUserOp {
	packed := &PackedUserOp{
		Sender:    uo.Sender,
		CallData:  uo.CallData,
		Signature: uo.Signature,
	}

	// Pack nonce
	if uo.Nonce != nil {
		packed.Nonce = (*big.Int)(uo.Nonce)
	} else {
		packed.Nonce = big.NewInt(0)
	}

	// Pack initCode (factory + factoryData if they exist)
	if uo.Factory != nil && len(uo.FactoryData) > 0 {
		initCode := make([]byte, 0, 20+len(uo.FactoryData))
		initCode = append(initCode, uo.Factory.Bytes()...)
		initCode = append(initCode, uo.FactoryData...)
		packed.InitCode = initCode
	} else {
		packed.InitCode = hexutil.Bytes{}
	}

	// Pack accountGasLimits (verificationGasLimit + callGasLimit, each 16 bytes with left padding)
	accountGasLimits := make([]byte, 32)
	if uo.VerificationGasLimit != nil {
		verificationBytes := (*big.Int)(uo.VerificationGasLimit).Bytes()
		copy(accountGasLimits[16-len(verificationBytes):16], verificationBytes)
	}
	if uo.CallGasLimit != nil {
		callBytes := (*big.Int)(uo.CallGasLimit).Bytes()
		copy(accountGasLimits[16+16-len(callBytes):32], callBytes)
	}
	packed.AccountGasLimits = accountGasLimits

	// Pack preVerificationGas as *big.Int
	if uo.PreVerificationGas != nil {
		packed.PreVerificationGas = (*big.Int)(uo.PreVerificationGas)
	} else {
		packed.PreVerificationGas = big.NewInt(0)
	}

	// Pack gasFees (maxPriorityFeePerGas + maxFeePerGas, each 16 bytes with left padding)
	gasFees := make([]byte, 32)
	if uo.MaxPriorityFeePerGas != nil {
		priorityBytes := (*big.Int)(uo.MaxPriorityFeePerGas).Bytes()
		copy(gasFees[16-len(priorityBytes):16], priorityBytes)
	}
	if uo.MaxFeePerGas != nil {
		maxFeeBytes := (*big.Int)(uo.MaxFeePerGas).Bytes()
		copy(gasFees[16+16-len(maxFeeBytes):32], maxFeeBytes)
	}
	packed.GasFees = gasFees

	// Pack paymasterAndData (paymaster + paymasterVerificationGasLimit + paymasterPostOpGasLimit + paymasterData)
	if uo.Paymaster != nil {
		// Calculate total size: 20 (paymaster) + 16 (verificationGasLimit) + 16 (postOpGasLimit) + paymasterData length
		paymasterAndData := make([]byte, 0, 52+len(uo.PaymasterData))

		// Add paymaster address
		paymasterAndData = append(paymasterAndData, uo.Paymaster.Bytes()...)

		// Add paymasterVerificationGasLimit (16 bytes with left padding)
		verificationLimit := make([]byte, 16)
		if uo.PaymasterVerificationGasLimit != nil {
			verificationBytes := (*big.Int)(uo.PaymasterVerificationGasLimit).Bytes()
			copy(verificationLimit[16-len(verificationBytes):16], verificationBytes)
		}
		paymasterAndData = append(paymasterAndData, verificationLimit...)

		// Add paymasterPostOpGasLimit (16 bytes with left padding)
		postOpLimit := make([]byte, 16)
		if uo.PaymasterPostOpGasLimit != nil {
			postOpBytes := (*big.Int)(uo.PaymasterPostOpGasLimit).Bytes()
			copy(postOpLimit[16-len(postOpBytes):16], postOpBytes)
		}
		paymasterAndData = append(paymasterAndData, postOpLimit...)

		// Add paymasterData (even if empty)
		paymasterAndData = append(paymasterAndData, uo.PaymasterData...)

		packed.PaymasterAndData = paymasterAndData
	} else {
		packed.PaymasterAndData = hexutil.Bytes{}
	}

	return packed
}

// getUserOpHashV07 computes the user operation hash for ERC-4337 v0.7
func (uo *UserOperation) GetUserOpHashV07(chainId *big.Int) (common.Hash, error) {
	packed := uo.PackUserOp()
	// Hash the initCode, callData, and paymasterAndData
	hashedInitCode := crypto.Keccak256Hash(packed.InitCode)
	hashedCallData := crypto.Keccak256Hash(packed.CallData)
	hashedPaymasterAndData := crypto.Keccak256Hash(packed.PaymasterAndData)

	// Use nonce directly as big.Int
	nonce := packed.Nonce
	if nonce == nil {
		nonce = big.NewInt(0)
	}

	// Create ABI types for encoding
	addressType, _ := abi.NewType("address", "", nil)
	uint256Type, _ := abi.NewType("uint256", "", nil)
	bytes32Type, _ := abi.NewType("bytes32", "", nil)

	// First level encoding: encode the user operation data
	userOpArgs := abi.Arguments{
		{Type: addressType}, // sender
		{Type: uint256Type}, // nonce
		{Type: bytes32Type}, // hashedInitCode
		{Type: bytes32Type}, // hashedCallData
		{Type: bytes32Type}, // accountGasLimits
		{Type: uint256Type}, // preVerificationGas
		{Type: bytes32Type}, // gasFees
		{Type: bytes32Type}, // hashedPaymasterAndData
	}

	// Convert accountGasLimits and gasFees to [32]byte for bytes32 encoding
	var accountGasLimits [32]byte
	copy(accountGasLimits[:], packed.AccountGasLimits)

	var gasFees [32]byte
	copy(gasFees[:], packed.GasFees)

	// Use preVerificationGas directly as *big.Int
	preVerificationGas := packed.PreVerificationGas
	if preVerificationGas == nil {
		preVerificationGas = big.NewInt(0)
	}

	userOpEncoded, err := userOpArgs.Pack(
		packed.Sender,
		nonce,
		hashedInitCode,
		hashedCallData,
		accountGasLimits,
		preVerificationGas,
		gasFees,
		hashedPaymasterAndData,
	)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to encode user operation: %v", err)
	}

	// Hash the encoded user operation
	userOpHash := crypto.Keccak256Hash(userOpEncoded)

	// Second level encoding: encode the hash with EntryPoint and chainId
	finalArgs := abi.Arguments{
		{Type: bytes32Type}, // userOpHash
		{Type: addressType}, // EntryPointV07
		{Type: uint256Type}, // chainId
	}

	finalEncoded, err := finalArgs.Pack(
		userOpHash,
		EntryPointV07,
		chainId,
	)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to encode final hash: %v", err)
	}

	// Return the final keccak256 hash
	return crypto.Keccak256Hash(finalEncoded), nil
}
