package server

import (
	"context"
	"errors"
	"math/big"
	"testing"
	"time"

	cfgTypes "github.com/0xPolygonHermez/zkevm-pool-manager/config/types"
	"github.com/0xPolygonHermez/zkevm-pool-manager/hex"
	"github.com/0xPolygonHermez/zkevm-pool-manager/types"
	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"
	mock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func NewMockConfig() Config {
	return Config{
		Host:                      "0.0.0.0",
		Port:                      8123,
		ReadTimeout:               cfgTypes.NewDuration(time.Second * 60),
		WriteTimeout:              cfgTypes.NewDuration(time.Second * 60),
		MaxRequestsPerIPAndSecond: 100,
	}
}

func TestSendRawTransaction(t *testing.T) {
	mockPoolDB := &poolDBMock{}
	mockSender := &senderMock{}
	cfg := NewMockConfig()
	errorAddTx := errors.New("failed to add tx to the pool")

	endpoints := NewEndpoints(cfg, mockPoolDB, mockSender)

	type testCase struct {
		Name          string
		RawTx         string
		Prepare       func(tc *testCase)
		SetupMocks    func()
		ExpectedError error
	}

	testCases := []testCase{
		{
			Name: "Send tx successfully",
			Prepare: func(tc *testCase) {
				tx := ethTypes.NewTransaction(1, common.HexToAddress("0x1"), big.NewInt(1), uint64(1), big.NewInt(1), []byte{})

				txBinary, err := tx.MarshalBinary()
				require.NoError(t, err)

				tc.RawTx = hex.EncodeToHex(txBinary)
				require.NoError(t, err)

			},
			SetupMocks: func() {
				mockPoolDB.On("AddTx", context.Background(), mock.IsType(types.L2Transaction{})).Return(nil).Once()
			},
			ExpectedError: nil,
		},
		{
			Name: "Send tx failed to add to the pool",
			Prepare: func(tc *testCase) {
				tx := ethTypes.NewTransaction(1, common.HexToAddress("0x1"), big.NewInt(1), uint64(1), big.NewInt(1), []byte{})

				txBinary, err := tx.MarshalBinary()
				require.NoError(t, err)

				tc.RawTx = hex.EncodeToHex(txBinary)
				require.NoError(t, err)
			},
			SetupMocks: func() {
				mockPoolDB.On("AddTx", context.Background(), mock.IsType(types.L2Transaction{})).Return(errorAddTx).Once()
			},
			ExpectedError: NewServerErrorWithData(DefaultErrorCode, errorAddTx.Error(), nil),
		},
		{
			Name: "Send invalid tx input",
			Prepare: func(tc *testCase) {
				tc.RawTx = "0x1234"
			},
			SetupMocks:    func() {},
			ExpectedError: NewServerErrorWithData(InvalidParamsErrorCode, "invalid tx input", nil),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			tc.Prepare(&tc)
			tc.SetupMocks()

			_, err := endpoints.SendRawTransaction(nil, tc.RawTx)
			assert.Equal(t, tc.ExpectedError, err)
		})
	}
}
