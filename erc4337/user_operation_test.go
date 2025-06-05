package erc4337

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper function to create address pointer
func addressPtr(addr string) *common.Address {
	a := common.HexToAddress(addr)
	return &a
}

// Helper function to create zero address pointer
func zeroAddressPtr() *common.Address {
	a := common.Address{}
	return &a
}

func TestUserOperation_MarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		userOp   *UserOperation
		expected map[string]interface{}
	}{
		{
			name: "complete user operation",
			userOp: &UserOperation{
				Sender:                        common.HexToAddress("0x1234567890123456789012345678901234567890"),
				Nonce:                         (*hexutil.Big)(big.NewInt(123)),
				Factory:                       addressPtr("0xabcdefabcdefabcdefabcdefabcdefabcdefabcd"),
				FactoryData:                   hexutil.MustDecode("0x1234"),
				CallData:                      hexutil.MustDecode("0x5678"),
				CallGasLimit:                  (*hexutil.Big)(big.NewInt(1000000)),
				VerificationGasLimit:          (*hexutil.Big)(big.NewInt(2000000)),
				PreVerificationGas:            (*hexutil.Big)(big.NewInt(3000000)),
				MaxPriorityFeePerGas:          (*hexutil.Big)(big.NewInt(1000000000)),
				MaxFeePerGas:                  (*hexutil.Big)(big.NewInt(2000000000)),
				Paymaster:                     addressPtr("0xfedcbafedcbafedcbafedcbafedcbafedcbafeda"),
				PaymasterVerificationGasLimit: (*hexutil.Big)(big.NewInt(500000)),
				PaymasterPostOpGasLimit:       (*hexutil.Big)(big.NewInt(100000)),
				PaymasterData:                 hexutil.MustDecode("0x9abc"),
				Signature:                     hexutil.MustDecode("0xdef0"),
			},
			expected: map[string]interface{}{
				"sender":                        "0x1234567890123456789012345678901234567890",
				"nonce":                         "0x000000000000000000000000000000000000000000000000000000000000007b",
				"factory":                       "0xabcdefabcdefabcdefabcdefabcdefabcdefabcd",
				"factoryData":                   "0x1234",
				"callData":                      "0x5678",
				"callGasLimit":                  "0xf4240",
				"verificationGasLimit":          "0x1e8480",
				"preVerificationGas":            "0x2dc6c0",
				"maxPriorityFeePerGas":          "0x3b9aca00",
				"maxFeePerGas":                  "0x77359400",
				"paymaster":                     "0xfedcbafedcbafedcbafedcbafedcbafedcbafeda",
				"paymasterVerificationGasLimit": "0x7a120",
				"paymasterPostOpGasLimit":       "0x186a0",
				"paymasterData":                 "0x9abc",
				"signature":                     "0xdef0",
			},
		},
		{
			name: "nil nonce should use zero padding",
			userOp: &UserOperation{
				Sender: common.HexToAddress("0x1234567890123456789012345678901234567890"),
				Nonce:  nil,
			},
			expected: map[string]interface{}{
				"nonce": "0x0000000000000000000000000000000000000000000000000000000000000000",
			},
		},
		{
			name: "zero nonce should use zero padding",
			userOp: &UserOperation{
				Sender: common.HexToAddress("0x1234567890123456789012345678901234567890"),
				Nonce:  (*hexutil.Big)(big.NewInt(0)),
			},
			expected: map[string]interface{}{
				"nonce": "0x0000000000000000000000000000000000000000000000000000000000000000",
			},
		},
		{
			name: "large nonce should be properly padded",
			userOp: &UserOperation{
				Sender: common.HexToAddress("0x1234567890123456789012345678901234567890"),
				Nonce:  (*hexutil.Big)(new(big.Int).SetBytes(common.Hex2Bytes("0123456789abcdef"))),
			},
			expected: map[string]interface{}{
				"nonce": "0x0000000000000000000000000000000000000000000000000123456789abcdef",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := tt.userOp.MarshalJSON()
			require.NoError(t, err)

			var result map[string]interface{}
			err = json.Unmarshal(data, &result)
			require.NoError(t, err)

			for key, expectedValue := range tt.expected {
				assert.Equal(t, expectedValue, result[key], "field %s mismatch", key)
			}
		})
	}
}

func TestUserOperation_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		jsonData string
		expected *UserOperation
		wantErr  bool
		errMsg   string
	}{
		{
			name: "complete user operation",
			jsonData: `{
				"sender": "0x1234567890123456789012345678901234567890",
				"nonce": "0x000000000000000000000000000000000000000000000000000000000000007b",
				"factory": "0xabcdefabcdefabcdefabcdefabcdefabcdefabcd",
				"factoryData": "0x1234",
				"callData": "0x5678",
				"callGasLimit": "0xf4240",
				"verificationGasLimit": "0x1e8480",
				"preVerificationGas": "0x2dc6c0",
				"maxPriorityFeePerGas": "0x3b9aca00",
				"maxFeePerGas": "0x77359400",
				"paymaster": "0xfedcbafedcbafedcbafedcbafedcbafedcbafeda",
				"paymasterVerificationGasLimit": "0x7a120",
				"paymasterPostOpGasLimit": "0x186a0",
				"paymasterData": "0x9abc",
				"signature": "0xdef0"
			}`,
			expected: &UserOperation{
				Sender:                        common.HexToAddress("0x1234567890123456789012345678901234567890"),
				Nonce:                         (*hexutil.Big)(big.NewInt(123)),
				Factory:                       addressPtr("0xabcdefabcdefabcdefabcdefabcdefabcdefabcd"),
				FactoryData:                   hexutil.MustDecode("0x1234"),
				CallData:                      hexutil.MustDecode("0x5678"),
				CallGasLimit:                  (*hexutil.Big)(big.NewInt(1000000)),
				VerificationGasLimit:          (*hexutil.Big)(big.NewInt(2000000)),
				PreVerificationGas:            (*hexutil.Big)(big.NewInt(3000000)),
				MaxPriorityFeePerGas:          (*hexutil.Big)(big.NewInt(1000000000)),
				MaxFeePerGas:                  (*hexutil.Big)(big.NewInt(2000000000)),
				Paymaster:                     addressPtr("0xfedcbafedcbafedcbafedcbafedcbafedcbafeda"),
				PaymasterVerificationGasLimit: (*hexutil.Big)(big.NewInt(500000)),
				PaymasterPostOpGasLimit:       (*hexutil.Big)(big.NewInt(100000)),
				PaymasterData:                 hexutil.MustDecode("0x9abc"),
				Signature:                     hexutil.MustDecode("0xdef0"),
			},
		},
		{
			name: "minimal user operation",
			jsonData: `{
				"sender": "0x1234567890123456789012345678901234567890",
				"nonce": "0x0",
				"factory": "0x0000000000000000000000000000000000000000",
				"factoryData": "0x",
				"callData": "0x",
				"callGasLimit": "0x0",
				"verificationGasLimit": "0x0",
				"preVerificationGas": "0x0",
				"maxPriorityFeePerGas": "0x0",
				"maxFeePerGas": "0x0",
				"paymaster": "0x0000000000000000000000000000000000000000",
				"paymasterVerificationGasLimit": "0x0",
				"paymasterPostOpGasLimit": "0x0",
				"paymasterData": "0x",
				"signature": "0x"
			}`,
			expected: &UserOperation{
				Sender:                        common.HexToAddress("0x1234567890123456789012345678901234567890"),
				Nonce:                         (*hexutil.Big)(big.NewInt(0)),
				Factory:                       zeroAddressPtr(),
				FactoryData:                   hexutil.Bytes{},
				CallData:                      hexutil.Bytes{},
				CallGasLimit:                  (*hexutil.Big)(big.NewInt(0)),
				VerificationGasLimit:          (*hexutil.Big)(big.NewInt(0)),
				PreVerificationGas:            (*hexutil.Big)(big.NewInt(0)),
				MaxPriorityFeePerGas:          (*hexutil.Big)(big.NewInt(0)),
				MaxFeePerGas:                  (*hexutil.Big)(big.NewInt(0)),
				Paymaster:                     zeroAddressPtr(),
				PaymasterVerificationGasLimit: (*hexutil.Big)(big.NewInt(0)),
				PaymasterPostOpGasLimit:       (*hexutil.Big)(big.NewInt(0)),
				PaymasterData:                 hexutil.Bytes{},
				Signature:                     hexutil.Bytes{},
			},
		},
		{
			name:     "invalid nonce",
			jsonData: `{"nonce": "invalid"}`,
			wantErr:  true,
			errMsg:   "invalid nonce",
		},
		{
			name:     "invalid callGasLimit",
			jsonData: `{"callGasLimit": "invalid"}`,
			wantErr:  true,
			errMsg:   "invalid callGasLimit",
		},
		{
			name:     "invalid JSON",
			jsonData: `{"incomplete": }`,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var userOp UserOperation
			err := json.Unmarshal([]byte(tt.jsonData), &userOp)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expected, &userOp)
		})
	}
}

func TestUserOperation_RoundTrip(t *testing.T) {
	original := &UserOperation{
		Sender:                        common.HexToAddress("0x1234567890123456789012345678901234567890"),
		Nonce:                         (*hexutil.Big)(big.NewInt(123456789)),
		Factory:                       addressPtr("0xabcdefabcdefabcdefabcdefabcdefabcdefabcd"),
		FactoryData:                   hexutil.MustDecode("0x1234abcd"),
		CallData:                      hexutil.MustDecode("0x5678ef90"),
		CallGasLimit:                  (*hexutil.Big)(big.NewInt(1000000)),
		VerificationGasLimit:          (*hexutil.Big)(big.NewInt(2000000)),
		PreVerificationGas:            (*hexutil.Big)(big.NewInt(3000000)),
		MaxPriorityFeePerGas:          (*hexutil.Big)(big.NewInt(1000000000)),
		MaxFeePerGas:                  (*hexutil.Big)(big.NewInt(2000000000)),
		Paymaster:                     addressPtr("0xfedcbafedcbafedcbafedcbafedcbafedcbafeda"),
		PaymasterVerificationGasLimit: (*hexutil.Big)(big.NewInt(500000)),
		PaymasterPostOpGasLimit:       (*hexutil.Big)(big.NewInt(100000)),
		PaymasterData:                 hexutil.MustDecode("0x9abcdef0"),
		Signature:                     hexutil.MustDecode("0xdef01234"),
	}

	// Marshal to JSON
	data, err := original.MarshalJSON()
	require.NoError(t, err)

	// Unmarshal from JSON
	var unmarshaled UserOperation
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	// Compare all fields
	assert.Equal(t, original.Sender, unmarshaled.Sender)
	assert.Equal(t, (*big.Int)(original.Nonce), (*big.Int)(unmarshaled.Nonce))
	assert.Equal(t, *original.Factory, *unmarshaled.Factory)
	assert.Equal(t, original.FactoryData, unmarshaled.FactoryData)
	assert.Equal(t, original.CallData, unmarshaled.CallData)
	assert.Equal(t, (*big.Int)(original.CallGasLimit), (*big.Int)(unmarshaled.CallGasLimit))
	assert.Equal(t, (*big.Int)(original.VerificationGasLimit), (*big.Int)(unmarshaled.VerificationGasLimit))
	assert.Equal(t, (*big.Int)(original.PreVerificationGas), (*big.Int)(unmarshaled.PreVerificationGas))
	assert.Equal(t, (*big.Int)(original.MaxPriorityFeePerGas), (*big.Int)(unmarshaled.MaxPriorityFeePerGas))
	assert.Equal(t, (*big.Int)(original.MaxFeePerGas), (*big.Int)(unmarshaled.MaxFeePerGas))
	assert.Equal(t, *original.Paymaster, *unmarshaled.Paymaster)
	assert.Equal(t, (*big.Int)(original.PaymasterVerificationGasLimit), (*big.Int)(unmarshaled.PaymasterVerificationGasLimit))
	assert.Equal(t, (*big.Int)(original.PaymasterPostOpGasLimit), (*big.Int)(unmarshaled.PaymasterPostOpGasLimit))
	assert.Equal(t, original.PaymasterData, unmarshaled.PaymasterData)
	assert.Equal(t, original.Signature, unmarshaled.Signature)
}

func TestUserOperation_NoncePadding(t *testing.T) {
	tests := []struct {
		name        string
		nonce       *big.Int
		expectedHex string
	}{
		{
			name:        "zero nonce",
			nonce:       big.NewInt(0),
			expectedHex: "0x0000000000000000000000000000000000000000000000000000000000000000",
		},
		{
			name:        "small nonce",
			nonce:       big.NewInt(123),
			expectedHex: "0x000000000000000000000000000000000000000000000000000000000000007b",
		},
		{
			name:        "large nonce",
			nonce:       new(big.Int).SetBytes(common.Hex2Bytes("0123456789abcdef")),
			expectedHex: "0x0000000000000000000000000000000000000000000000000123456789abcdef",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userOp := &UserOperation{
				Sender: common.HexToAddress("0x1234567890123456789012345678901234567890"),
				Nonce:  (*hexutil.Big)(tt.nonce),
			}

			data, err := userOp.MarshalJSON()
			require.NoError(t, err)

			var result map[string]interface{}
			err = json.Unmarshal(data, &result)
			require.NoError(t, err)

			assert.Equal(t, tt.expectedHex, result["nonce"])
		})
	}
}

func TestUserOperation_PackUserOp(t *testing.T) {
	tests := []struct {
		name     string
		userOp   *UserOperation
		expected *PackedUserOp
	}{
		{
			name: "complete user operation with all fields",
			userOp: &UserOperation{
				Sender:                        common.HexToAddress("0x1234567890123456789012345678901234567890"),
				Nonce:                         (*hexutil.Big)(big.NewInt(123)),
				Factory:                       addressPtr("0xabcdefabcdefabcdefabcdefabcdefabcdefabcd"),
				FactoryData:                   hexutil.MustDecode("0x1234"),
				CallData:                      hexutil.MustDecode("0x5678"),
				CallGasLimit:                  (*hexutil.Big)(big.NewInt(1000000)),
				VerificationGasLimit:          (*hexutil.Big)(big.NewInt(2000000)),
				PreVerificationGas:            (*hexutil.Big)(big.NewInt(3000000)),
				MaxPriorityFeePerGas:          (*hexutil.Big)(big.NewInt(1000000000)),
				MaxFeePerGas:                  (*hexutil.Big)(big.NewInt(2000000000)),
				Paymaster:                     addressPtr("0xfedcbafedcbafedcbafedcbafedcbafedcbafeda"),
				PaymasterVerificationGasLimit: (*hexutil.Big)(big.NewInt(500000)),
				PaymasterPostOpGasLimit:       (*hexutil.Big)(big.NewInt(100000)),
				PaymasterData:                 hexutil.MustDecode("0x9abc"),
				Signature:                     hexutil.MustDecode("0xdef0"),
			},
			expected: &PackedUserOp{
				Sender:   common.HexToAddress("0x1234567890123456789012345678901234567890"),
				Nonce:    big.NewInt(123),
				InitCode: append(common.HexToAddress("0xabcdefabcdefabcdefabcdefabcdefabcdefabcd").Bytes(), hexutil.MustDecode("0x1234")...),
				CallData: hexutil.MustDecode("0x5678"),
				AccountGasLimits: func() hexutil.Bytes {
					limits := make([]byte, 32)
					// VerificationGasLimit (2000000) in first 16 bytes
					verificationBytes := big.NewInt(2000000).Bytes()
					copy(limits[16-len(verificationBytes):16], verificationBytes)
					// CallGasLimit (1000000) in last 16 bytes
					callBytes := big.NewInt(1000000).Bytes()
					copy(limits[32-len(callBytes):32], callBytes)
					return limits
				}(),
				PreVerificationGas: big.NewInt(3000000),
				GasFees: func() hexutil.Bytes {
					fees := make([]byte, 32)
					// MaxPriorityFeePerGas (1000000000) in first 16 bytes
					priorityBytes := big.NewInt(1000000000).Bytes()
					copy(fees[16-len(priorityBytes):16], priorityBytes)
					// MaxFeePerGas (2000000000) in last 16 bytes
					maxFeeBytes := big.NewInt(2000000000).Bytes()
					copy(fees[32-len(maxFeeBytes):32], maxFeeBytes)
					return fees
				}(),
				PaymasterAndData: func() hexutil.Bytes {
					data := make([]byte, 0, 52+2) // 20 + 16 + 16 + 2 bytes for paymasterData
					// Paymaster address (20 bytes)
					data = append(data, common.HexToAddress("0xfedcbafedcbafedcbafedcbafedcbafedcbafeda").Bytes()...)
					// PaymasterVerificationGasLimit (16 bytes)
					verificationLimit := make([]byte, 16)
					verificationBytes := big.NewInt(500000).Bytes()
					copy(verificationLimit[16-len(verificationBytes):16], verificationBytes)
					data = append(data, verificationLimit...)
					// PaymasterPostOpGasLimit (16 bytes)
					postOpLimit := make([]byte, 16)
					postOpBytes := big.NewInt(100000).Bytes()
					copy(postOpLimit[16-len(postOpBytes):16], postOpBytes)
					data = append(data, postOpLimit...)
					// PaymasterData
					data = append(data, hexutil.MustDecode("0x9abc")...)
					return data
				}(),
				Signature: hexutil.MustDecode("0xdef0"),
			},
		},
		{
			name: "minimal user operation without factory and paymaster",
			userOp: &UserOperation{
				Sender:               common.HexToAddress("0x1234567890123456789012345678901234567890"),
				Nonce:                (*hexutil.Big)(big.NewInt(0)),
				Factory:              nil,
				FactoryData:          hexutil.Bytes{},
				CallData:             hexutil.MustDecode("0x5678"),
				CallGasLimit:         (*hexutil.Big)(big.NewInt(100000)),
				VerificationGasLimit: (*hexutil.Big)(big.NewInt(200000)),
				PreVerificationGas:   (*hexutil.Big)(big.NewInt(50000)),
				MaxPriorityFeePerGas: (*hexutil.Big)(big.NewInt(1000000)),
				MaxFeePerGas:         (*hexutil.Big)(big.NewInt(2000000)),
				Paymaster:            nil,
				PaymasterData:        hexutil.Bytes{},
				Signature:            hexutil.MustDecode("0xabcd"),
			},
			expected: &PackedUserOp{
				Sender:   common.HexToAddress("0x1234567890123456789012345678901234567890"),
				Nonce:    big.NewInt(0),
				InitCode: hexutil.Bytes{},
				CallData: hexutil.MustDecode("0x5678"),
				AccountGasLimits: func() hexutil.Bytes {
					limits := make([]byte, 32)
					// VerificationGasLimit (200000) in first 16 bytes
					verificationBytes := big.NewInt(200000).Bytes()
					copy(limits[16-len(verificationBytes):16], verificationBytes)
					// CallGasLimit (100000) in last 16 bytes
					callBytes := big.NewInt(100000).Bytes()
					copy(limits[32-len(callBytes):32], callBytes)
					return limits
				}(),
				PreVerificationGas: big.NewInt(50000),
				GasFees: func() hexutil.Bytes {
					fees := make([]byte, 32)
					// MaxPriorityFeePerGas (1000000) in first 16 bytes
					priorityBytes := big.NewInt(1000000).Bytes()
					copy(fees[16-len(priorityBytes):16], priorityBytes)
					// MaxFeePerGas (2000000) in last 16 bytes
					maxFeeBytes := big.NewInt(2000000).Bytes()
					copy(fees[32-len(maxFeeBytes):32], maxFeeBytes)
					return fees
				}(),
				PaymasterAndData: hexutil.Bytes{},
				Signature:        hexutil.MustDecode("0xabcd"),
			},
		},
		{
			name: "user operation with factory but no paymaster",
			userOp: &UserOperation{
				Sender:               common.HexToAddress("0x1234567890123456789012345678901234567890"),
				Nonce:                (*hexutil.Big)(big.NewInt(456)),
				Factory:              addressPtr("0xabcdefabcdefabcdefabcdefabcdefabcdefabcd"),
				FactoryData:          hexutil.MustDecode("0xfacade"),
				CallData:             hexutil.MustDecode("0x5678"),
				CallGasLimit:         (*hexutil.Big)(big.NewInt(100000)),
				VerificationGasLimit: (*hexutil.Big)(big.NewInt(200000)),
				PreVerificationGas:   (*hexutil.Big)(big.NewInt(50000)),
				MaxPriorityFeePerGas: (*hexutil.Big)(big.NewInt(1000000)),
				MaxFeePerGas:         (*hexutil.Big)(big.NewInt(2000000)),
				Paymaster:            nil,
				PaymasterData:        hexutil.Bytes{},
				Signature:            hexutil.MustDecode("0xabcd"),
			},
			expected: &PackedUserOp{
				Sender:   common.HexToAddress("0x1234567890123456789012345678901234567890"),
				Nonce:    big.NewInt(456),
				InitCode: append(common.HexToAddress("0xabcdefabcdefabcdefabcdefabcdefabcdefabcd").Bytes(), hexutil.MustDecode("0xfacade")...),
				CallData: hexutil.MustDecode("0x5678"),
				AccountGasLimits: func() hexutil.Bytes {
					limits := make([]byte, 32)
					verificationBytes := big.NewInt(200000).Bytes()
					copy(limits[16-len(verificationBytes):16], verificationBytes)
					callBytes := big.NewInt(100000).Bytes()
					copy(limits[32-len(callBytes):32], callBytes)
					return limits
				}(),
				PreVerificationGas: big.NewInt(50000),
				GasFees: func() hexutil.Bytes {
					fees := make([]byte, 32)
					priorityBytes := big.NewInt(1000000).Bytes()
					copy(fees[16-len(priorityBytes):16], priorityBytes)
					maxFeeBytes := big.NewInt(2000000).Bytes()
					copy(fees[32-len(maxFeeBytes):32], maxFeeBytes)
					return fees
				}(),
				PaymasterAndData: hexutil.Bytes{},
				Signature:        hexutil.MustDecode("0xabcd"),
			},
		},
		{
			name: "user operation with nil values",
			userOp: &UserOperation{
				Sender:               common.HexToAddress("0x1234567890123456789012345678901234567890"),
				Nonce:                nil,
				Factory:              nil,
				FactoryData:          hexutil.Bytes{},
				CallData:             hexutil.MustDecode("0x5678"),
				CallGasLimit:         nil,
				VerificationGasLimit: nil,
				PreVerificationGas:   nil,
				MaxPriorityFeePerGas: nil,
				MaxFeePerGas:         nil,
				Paymaster:            nil,
				PaymasterData:        hexutil.Bytes{},
				Signature:            hexutil.MustDecode("0xabcd"),
			},
			expected: &PackedUserOp{
				Sender:             common.HexToAddress("0x1234567890123456789012345678901234567890"),
				Nonce:              big.NewInt(0),
				InitCode:           hexutil.Bytes{},
				CallData:           hexutil.MustDecode("0x5678"),
				AccountGasLimits:   make([]byte, 32), // all zeros
				PreVerificationGas: big.NewInt(0),    // zero big.Int
				GasFees:            make([]byte, 32), // all zeros
				PaymasterAndData:   hexutil.Bytes{},
				Signature:          hexutil.MustDecode("0xabcd"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			packed := tt.userOp.PackUserOp()

			assert.Equal(t, tt.expected.Sender, packed.Sender)
			assert.Equal(t, tt.expected.Nonce, packed.Nonce)
			assert.Equal(t, tt.expected.InitCode, packed.InitCode)
			assert.Equal(t, tt.expected.CallData, packed.CallData)
			assert.Equal(t, tt.expected.AccountGasLimits, packed.AccountGasLimits)
			assert.Equal(t, tt.expected.PreVerificationGas, packed.PreVerificationGas)
			assert.Equal(t, tt.expected.GasFees, packed.GasFees)
			assert.Equal(t, tt.expected.PaymasterAndData, packed.PaymasterAndData)
			assert.Equal(t, tt.expected.Signature, packed.Signature)
		})
	}
}

func TestUserOperation_PackUserOp_ByteOrdering(t *testing.T) {
	// Test specific byte ordering for gas limits and fees
	userOp := &UserOperation{
		Sender:               common.HexToAddress("0x1234567890123456789012345678901234567890"),
		Nonce:                (*hexutil.Big)(big.NewInt(1)),
		CallGasLimit:         (*hexutil.Big)(big.NewInt(0x123456)), // 1193046
		VerificationGasLimit: (*hexutil.Big)(big.NewInt(0x789abc)), // 7903932
		PreVerificationGas:   (*hexutil.Big)(big.NewInt(0xdef012)), // 14610450
		MaxPriorityFeePerGas: (*hexutil.Big)(big.NewInt(0x345678)), // 3430008
		MaxFeePerGas:         (*hexutil.Big)(big.NewInt(0x9abcde)), // 10141918
		CallData:             hexutil.Bytes{},
		Signature:            hexutil.Bytes{},
	}

	packed := userOp.PackUserOp()

	// Verify AccountGasLimits byte ordering
	expectedAccountGasLimits := make([]byte, 32)
	// VerificationGasLimit in first 16 bytes (big endian)
	copy(expectedAccountGasLimits[13:16], []byte{0x78, 0x9a, 0xbc})
	// CallGasLimit in last 16 bytes (big endian)
	copy(expectedAccountGasLimits[29:32], []byte{0x12, 0x34, 0x56})
	assert.Equal(t, expectedAccountGasLimits, []byte(packed.AccountGasLimits))

	// Verify PreVerificationGas byte ordering
	// Verify PreVerificationGas
	assert.Equal(t, big.NewInt(0xdef012), packed.PreVerificationGas)

	// Verify GasFees byte ordering
	expectedGasFees := make([]byte, 32)
	// MaxPriorityFeePerGas in first 16 bytes
	copy(expectedGasFees[13:16], []byte{0x34, 0x56, 0x78})
	// MaxFeePerGas in last 16 bytes
	copy(expectedGasFees[29:32], []byte{0x9a, 0xbc, 0xde})
	assert.Equal(t, expectedGasFees, []byte(packed.GasFees))
}

func TestGetUserOpHashV07(t *testing.T) {
	tests := []struct {
		name        string
		userOp      *UserOperation
		chainId     *big.Int
		wantErr     bool
		errContains string
	}{
		{
			name: "complete user operation",
			userOp: &UserOperation{
				Sender:                        common.HexToAddress("0x1234567890123456789012345678901234567890"),
				Nonce:                         (*hexutil.Big)(big.NewInt(123)),
				Factory:                       addressPtr("0xabcdefabcdefabcdefabcdefabcdefabcdefabcd"),
				FactoryData:                   hexutil.MustDecode("0x1234"),
				CallData:                      hexutil.MustDecode("0x5678"),
				CallGasLimit:                  (*hexutil.Big)(big.NewInt(1000000)),
				VerificationGasLimit:          (*hexutil.Big)(big.NewInt(2000000)),
				PreVerificationGas:            (*hexutil.Big)(big.NewInt(3000000)),
				MaxPriorityFeePerGas:          (*hexutil.Big)(big.NewInt(1000000000)),
				MaxFeePerGas:                  (*hexutil.Big)(big.NewInt(2000000000)),
				Paymaster:                     addressPtr("0xfedcbafedcbafedcbafedcbafedcbafedcbafeda"),
				PaymasterVerificationGasLimit: (*hexutil.Big)(big.NewInt(500000)),
				PaymasterPostOpGasLimit:       (*hexutil.Big)(big.NewInt(100000)),
				PaymasterData:                 hexutil.MustDecode("0x9abc"),
				Signature:                     hexutil.MustDecode("0xdef0"),
			},
			chainId: big.NewInt(1), // Ethereum mainnet
			wantErr: false,
		},
		{
			name: "minimal user operation",
			userOp: &UserOperation{
				Sender:               common.HexToAddress("0x1234567890123456789012345678901234567890"),
				Nonce:                (*hexutil.Big)(big.NewInt(0)),
				Factory:              nil,
				FactoryData:          hexutil.Bytes{},
				CallData:             hexutil.MustDecode("0x5678"),
				CallGasLimit:         (*hexutil.Big)(big.NewInt(100000)),
				VerificationGasLimit: (*hexutil.Big)(big.NewInt(200000)),
				PreVerificationGas:   (*hexutil.Big)(big.NewInt(50000)),
				MaxPriorityFeePerGas: (*hexutil.Big)(big.NewInt(1000000)),
				MaxFeePerGas:         (*hexutil.Big)(big.NewInt(2000000)),
				Paymaster:            nil,
				PaymasterData:        hexutil.Bytes{},
				Signature:            hexutil.MustDecode("0xabcd"),
			},
			chainId: big.NewInt(137), // Polygon
			wantErr: false,
		},
		{
			name: "different chain id",
			userOp: &UserOperation{
				Sender:               common.HexToAddress("0x1234567890123456789012345678901234567890"),
				Nonce:                (*hexutil.Big)(big.NewInt(1)),
				Factory:              nil,
				FactoryData:          hexutil.Bytes{},
				CallData:             hexutil.MustDecode("0x1234"),
				CallGasLimit:         (*hexutil.Big)(big.NewInt(100000)),
				VerificationGasLimit: (*hexutil.Big)(big.NewInt(200000)),
				PreVerificationGas:   (*hexutil.Big)(big.NewInt(50000)),
				MaxPriorityFeePerGas: (*hexutil.Big)(big.NewInt(1000000)),
				MaxFeePerGas:         (*hexutil.Big)(big.NewInt(2000000)),
				Paymaster:            nil,
				PaymasterData:        hexutil.Bytes{},
				Signature:            hexutil.MustDecode("0x5678"),
			},
			chainId: big.NewInt(42161), // Arbitrum One
			wantErr: false,
		},
		{
			name: "large nonce value",
			userOp: &UserOperation{
				Sender:               common.HexToAddress("0x1234567890123456789012345678901234567890"),
				Nonce:                (*hexutil.Big)(new(big.Int).SetBytes(common.Hex2Bytes("1234567890abcdef"))),
				Factory:              nil,
				FactoryData:          hexutil.Bytes{},
				CallData:             hexutil.MustDecode("0x1234"),
				CallGasLimit:         (*hexutil.Big)(big.NewInt(100000)),
				VerificationGasLimit: (*hexutil.Big)(big.NewInt(200000)),
				PreVerificationGas:   (*hexutil.Big)(big.NewInt(50000)),
				MaxPriorityFeePerGas: (*hexutil.Big)(big.NewInt(1000000)),
				MaxFeePerGas:         (*hexutil.Big)(big.NewInt(2000000)),
				Paymaster:            nil,
				PaymasterData:        hexutil.Bytes{},
				Signature:            hexutil.MustDecode("0x5678"),
			},
			chainId: big.NewInt(1),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := tt.userOp.GetUserOpHashV07(tt.chainId)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			require.NoError(t, err)
			// Verify the hash is not empty
			assert.NotEqual(t, common.Hash{}, hash)
			// Verify the hash is 32 bytes
			assert.Equal(t, 32, len(hash))
		})
	}
}

func TestGetUserOpHashV07_Consistency(t *testing.T) {
	// Test that the same input produces the same hash
	userOp := &UserOperation{
		Sender:               common.HexToAddress("0x1234567890123456789012345678901234567890"),
		Nonce:                (*hexutil.Big)(big.NewInt(1)),
		Factory:              nil,
		FactoryData:          hexutil.Bytes{},
		CallData:             hexutil.MustDecode("0x1234"),
		CallGasLimit:         (*hexutil.Big)(big.NewInt(100000)),
		VerificationGasLimit: (*hexutil.Big)(big.NewInt(200000)),
		PreVerificationGas:   (*hexutil.Big)(big.NewInt(50000)),
		MaxPriorityFeePerGas: (*hexutil.Big)(big.NewInt(1000000)),
		MaxFeePerGas:         (*hexutil.Big)(big.NewInt(2000000)),
		Paymaster:            nil,
		PaymasterData:        hexutil.Bytes{},
		Signature:            hexutil.MustDecode("0x5678"),
	}
	chainId := big.NewInt(1)

	hash1, err1 := userOp.GetUserOpHashV07(chainId)
	require.NoError(t, err1)

	hash2, err2 := userOp.GetUserOpHashV07(chainId)
	require.NoError(t, err2)

	assert.Equal(t, hash1, hash2, "Same input should produce same hash")
}

func TestGetUserOpHashV07_DifferentInputs(t *testing.T) {
	// Test that different inputs produce different hashes
	baseOp := &UserOperation{
		Sender:               common.HexToAddress("0x1234567890123456789012345678901234567890"),
		Nonce:                (*hexutil.Big)(big.NewInt(1)),
		Factory:              nil,
		FactoryData:          hexutil.Bytes{},
		CallData:             hexutil.MustDecode("0x1234"),
		CallGasLimit:         (*hexutil.Big)(big.NewInt(100000)),
		VerificationGasLimit: (*hexutil.Big)(big.NewInt(200000)),
		PreVerificationGas:   (*hexutil.Big)(big.NewInt(50000)),
		MaxPriorityFeePerGas: (*hexutil.Big)(big.NewInt(1000000)),
		MaxFeePerGas:         (*hexutil.Big)(big.NewInt(2000000)),
		Paymaster:            nil,
		PaymasterData:        hexutil.Bytes{},
		Signature:            hexutil.MustDecode("0x5678"),
	}
	chainId := big.NewInt(1)

	baseHash, err := baseOp.GetUserOpHashV07(chainId)
	require.NoError(t, err)

	// Test different sender
	diffSenderOp := *baseOp
	diffSenderOp.Sender = common.HexToAddress("0x9876543210987654321098765432109876543210")
	diffSenderHash, err := diffSenderOp.GetUserOpHashV07(chainId)
	require.NoError(t, err)
	assert.NotEqual(t, baseHash, diffSenderHash, "Different sender should produce different hash")

	// Test different nonce
	diffNonceOp := *baseOp
	diffNonceOp.Nonce = (*hexutil.Big)(big.NewInt(2))
	diffNonceHash, err := diffNonceOp.GetUserOpHashV07(chainId)
	require.NoError(t, err)
	assert.NotEqual(t, baseHash, diffNonceHash, "Different nonce should produce different hash")

	// Test different chain ID
	diffChainHash, err := baseOp.GetUserOpHashV07(big.NewInt(137))
	require.NoError(t, err)
	assert.NotEqual(t, baseHash, diffChainHash, "Different chain ID should produce different hash")

	// Test different call data
	diffCallDataOp := *baseOp
	diffCallDataOp.CallData = hexutil.MustDecode("0x9876")
	diffCallDataHash, err := diffCallDataOp.GetUserOpHashV07(chainId)
	require.NoError(t, err)
	assert.NotEqual(t, baseHash, diffCallDataHash, "Different call data should produce different hash")
}
