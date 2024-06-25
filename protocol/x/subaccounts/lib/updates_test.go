package lib_test

import (
	"math/big"
	"testing"

	"github.com/dydxprotocol/v4-chain/protocol/dtypes"
	"github.com/dydxprotocol/v4-chain/protocol/lib/margin"
	assettypes "github.com/dydxprotocol/v4-chain/protocol/x/assets/types"
	perptypes "github.com/dydxprotocol/v4-chain/protocol/x/perpetuals/types"
	pricetypes "github.com/dydxprotocol/v4-chain/protocol/x/prices/types"
	"github.com/dydxprotocol/v4-chain/protocol/x/subaccounts/lib"
	"github.com/dydxprotocol/v4-chain/protocol/x/subaccounts/types"
	"github.com/stretchr/testify/require"
)

func TestIsValidStateTransitionForUndercollateralizedSubaccount_ZeroMarginRequirements(t *testing.T) {
	tests := map[string]struct {
		oldNC  *big.Int
		oldIMR *big.Int
		oldMMR *big.Int
		newNC  *big.Int
		newMMR *big.Int

		expectedResult types.UpdateResult
	}{
		// Tests when current margin requirement is zero and margin requirement increases.
		"fails when MMR increases and TNC decreases - negative TNC": {
			oldNC:          big.NewInt(-1),
			oldIMR:         big.NewInt(0),
			oldMMR:         big.NewInt(0),
			newNC:          big.NewInt(-2),
			newMMR:         big.NewInt(1),
			expectedResult: types.StillUndercollateralized,
		},
		"fails when MMR increases and TNC stays the same - negative TNC": {
			oldNC:          big.NewInt(-1),
			oldIMR:         big.NewInt(0),
			oldMMR:         big.NewInt(0),
			newNC:          big.NewInt(-1),
			newMMR:         big.NewInt(1),
			expectedResult: types.StillUndercollateralized,
		},
		"fails when MMR increases and TNC increases - negative TNC": {
			oldNC:          big.NewInt(-1),
			oldIMR:         big.NewInt(0),
			oldMMR:         big.NewInt(0),
			newNC:          big.NewInt(100),
			newMMR:         big.NewInt(1),
			expectedResult: types.StillUndercollateralized,
		},
		// Tests when both margin requirements are zero.
		"fails when both new and old MMR are zero and TNC stays the same": {
			oldNC:          big.NewInt(-1),
			oldIMR:         big.NewInt(0),
			oldMMR:         big.NewInt(0),
			newNC:          big.NewInt(-1),
			newMMR:         big.NewInt(0),
			expectedResult: types.StillUndercollateralized,
		},
		"fails when both new and old MMR are zero and TNC decrease from negative to negative": {
			oldNC:          big.NewInt(-1),
			oldIMR:         big.NewInt(0),
			oldMMR:         big.NewInt(0),
			newNC:          big.NewInt(-2),
			newMMR:         big.NewInt(0),
			expectedResult: types.StillUndercollateralized,
		},
		"succeeds when both new and old MMR are zero and TNC increases": {
			oldNC:          big.NewInt(-2),
			oldIMR:         big.NewInt(0),
			oldMMR:         big.NewInt(0),
			newNC:          big.NewInt(-1),
			newMMR:         big.NewInt(0),
			expectedResult: types.Success,
		},
		// Tests when new margin requirement is zero.
		"fails when MMR decreased to zero, and TNC increases but is still negative": {
			oldNC:          big.NewInt(-2),
			oldIMR:         big.NewInt(1),
			oldMMR:         big.NewInt(1),
			newNC:          big.NewInt(-1),
			newMMR:         big.NewInt(0),
			expectedResult: types.StillUndercollateralized,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			require.Equal(
				t,
				tc.expectedResult,
				lib.IsValidStateTransitionForUndercollateralizedSubaccount(
					margin.Risk{
						NC:  tc.oldNC,
						IMR: tc.oldIMR,
						MMR: tc.oldMMR,
					},
					margin.Risk{
						NC:  tc.newNC,
						MMR: tc.newMMR,
					},
				),
			)
		})
	}
}

func TestGetRiskForSubaccount(t *testing.T) {
	subaccountId := types.SubaccountId{Owner: "test", Number: 1}
	tests := map[string]struct {
		settledUpdate types.SettledUpdate
		perpInfos     perptypes.PerpInfos
		expectedRisk  margin.Risk
		expectedErr   error
	}{
		"no account": {
			settledUpdate: types.SettledUpdate{},
			perpInfos:     perptypes.PerpInfos{},
			expectedRisk:  margin.ZeroRisk(),
			expectedErr:   nil,
		},
		"no updates": {
			settledUpdate: types.SettledUpdate{
				SettledSubaccount: types.Subaccount{
					Id: &subaccountId,
					PerpetualPositions: []*types.PerpetualPosition{
						createPerpPosition(1, big.NewInt(100), big.NewInt(0)),
					},
					AssetPositions: createUsdcAmount(big.NewInt(100)),
				},
				PerpetualUpdates: []types.PerpetualUpdate{},
				AssetUpdates:     []types.AssetUpdate{},
			},
			perpInfos: perptypes.PerpInfos{
				1: createPerpInfo(1, -6, 100, 0),
			},
			expectedRisk: margin.Risk{
				NC:  big.NewInt(100*100 + 100),
				IMR: big.NewInt(100 * 100 * 0.1),
				MMR: big.NewInt(100 * 100 * 0.1 * 0.5),
			},
			expectedErr: nil,
		},
		"one update": {
			settledUpdate: types.SettledUpdate{
				SettledSubaccount: types.Subaccount{
					Id: &subaccountId,
					PerpetualPositions: []*types.PerpetualPosition{
						createPerpPosition(1, big.NewInt(100), big.NewInt(0)),
					},
					AssetPositions: createUsdcAmount(big.NewInt(100)),
				},
				PerpetualUpdates: []types.PerpetualUpdate{
					{
						PerpetualId:      2,
						BigQuantumsDelta: big.NewInt(-25),
					},
				},
				AssetUpdates: []types.AssetUpdate{
					{
						AssetId:          assettypes.AssetUsdc.Id,
						BigQuantumsDelta: big.NewInt(10),
					},
				},
			},
			perpInfos: perptypes.PerpInfos{
				1: createPerpInfo(1, -6, 100, 0),
				2: createPerpInfo(2, -6, 200, 0),
			},
			expectedRisk: margin.Risk{
				NC:  big.NewInt((100*100 + 100) + (-25*200 + 10)),
				IMR: big.NewInt((100 * 100 * 0.1) + (25 * 200 * 0.1)),
				MMR: big.NewInt((100 * 100 * 0.1 * 0.5) + (25 * 200 * 0.1 * 0.5)),
			},
			expectedErr: nil,
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			risk, err := lib.GetRiskForSubaccount(tc.settledUpdate, tc.perpInfos)
			require.Equal(t, tc.expectedRisk, risk)
			if tc.expectedErr != nil {
				require.Equal(t, tc.expectedErr, err)
			} else {
				require.Nil(t, err)
			}
		})
	}
}

func TestGetRiskForSubaccount_Panic(t *testing.T) {
	sa := types.SettledUpdate{
		SettledSubaccount: types.Subaccount{
			Id: &types.SubaccountId{Owner: "test", Number: 1},
			PerpetualPositions: []*types.PerpetualPosition{
				createPerpPosition(1, big.NewInt(100), big.NewInt(0)),
			},
			AssetPositions: createUsdcAmount(big.NewInt(100)),
		},
		PerpetualUpdates: []types.PerpetualUpdate{},
		AssetUpdates:     []types.AssetUpdate{},
	}
	emptyPerpInfos := perptypes.PerpInfos{}

	// Panics since relevant perpetual information cannot be found.
	require.Panics(t, func() {
		_, _ = lib.GetRiskForSubaccount(sa, emptyPerpInfos)
	})
}

func createPerpPosition(
	id uint32,
	quantums *big.Int,
	fundingIndex *big.Int,
) *types.PerpetualPosition {
	return &types.PerpetualPosition{
		PerpetualId:  id,
		Quantums:     dtypes.NewIntFromBigInt(quantums),
		FundingIndex: dtypes.NewIntFromBigInt(fundingIndex),
	}
}

func createUsdcAmount(amount *big.Int) []*types.AssetPosition {
	return []*types.AssetPosition{
		{
			AssetId:  assettypes.AssetUsdc.Id,
			Quantums: dtypes.NewIntFromBigInt(amount),
		},
	}
}

func createPerpInfo(
	id uint32,
	atomicResolution int32,
	price uint64,
	priceExponent int32,
) perptypes.PerpInfo {
	return perptypes.PerpInfo{
		Perpetual: perptypes.Perpetual{
			Params: perptypes.PerpetualParams{
				Id:               id,
				Ticker:           "test ticker",
				MarketId:         id,
				AtomicResolution: atomicResolution,
				LiquidityTier:    id,
			},
			FundingIndex: dtypes.NewInt(0),
			OpenInterest: dtypes.NewInt(0),
		},
		Price: pricetypes.MarketPrice{
			Id:       id,
			Exponent: priceExponent,
			Price:    price,
		},
		LiquidityTier: perptypes.LiquidityTier{
			Id:                     id,
			InitialMarginPpm:       100_000,
			MaintenanceFractionPpm: 500_000,
			OpenInterestLowerCap:   0,
			OpenInterestUpperCap:   0,
		},
	}
}