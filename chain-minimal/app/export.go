package app

import (
	"encoding/json"
	"fmt"

	cmtproto "github.com/cometbft/cometbft/api/cometbft/types/v1"
	cmttypes "github.com/cometbft/cometbft/types"

	"cosmossdk.io/collections"
	storetypes "cosmossdk.io/store/types"
	"cosmossdk.io/x/staking"
	stakingtypes "cosmossdk.io/x/staking/types"

	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// ExportAppStateAndValidators exports the state of the application for a genesis file.
func (app *MiniApp) ExportAppStateAndValidators(
	forZeroHeight bool,
	jailAllowedAddrs,
	modulesToExport []string,
) (servertypes.ExportedApp, error) {
	// as if they could withdraw from the start of the next block
	ctx := app.NewContextLegacy(true, cmtproto.Header{Height: app.LastBlockHeight()})

	// We export at last height + 1, because that's the height at which
	// CometBFT will start InitChain.
	height := app.LastBlockHeight() + 1
	if forZeroHeight {
		height = 0
		app.prepForZeroHeightGenesis(ctx, jailAllowedAddrs)
	}

	genState, err := app.ModuleManager.ExportGenesisForModules(ctx, modulesToExport)
	if err != nil {
		return servertypes.ExportedApp{}, err
	}

	appState, err := json.MarshalIndent(genState, "", "  ")
	if err != nil {
		return servertypes.ExportedApp{}, err
	}

	validators, err := staking.WriteValidators(ctx, app.StakingKeeper)
	cmtValidators := []cmttypes.GenesisValidator{}
	for _, val := range validators {
		cmtPk, err := cryptocodec.ToCmtPubKeyInterface(val.PubKey)
		if err != nil {
			return servertypes.ExportedApp{}, err
		}
		cmtVal := cmttypes.GenesisValidator{
			Address: val.Address.Bytes(),
			PubKey:  cmtPk,
			Power:   val.Power,
			Name:    val.Name,
		}

		cmtValidators = append(cmtValidators, cmtVal)
	}

	return servertypes.ExportedApp{
		AppState:        appState,
		Validators:      cmtValidators,
		Height:          height,
		ConsensusParams: app.BaseApp.GetConsensusParams(ctx),
	}, err
}

// prepForZeroHeightGenesis prepares for fresh start at zero height
// NOTE zero height genesis is a temporary feature which will be deprecated
//
//	in favor of export at a block height
func (app *MiniApp) prepForZeroHeightGenesis(ctx sdk.Context, jailAllowedAddrs []string) {
	applyAllowedAddrs := false

	// check if there is a allowed address list
	if len(jailAllowedAddrs) > 0 {
		applyAllowedAddrs = true
	}

	allowedAddrsMap := make(map[string]bool)

	for _, addr := range jailAllowedAddrs {
		_, err := app.interfaceRegistry.SigningContext().ValidatorAddressCodec().StringToBytes(addr)
		if err != nil {
			panic(err)
		}
		allowedAddrsMap[addr] = true
	}

	/* Handle fee distribution state. */

	// withdraw all validator commission
	err := app.StakingKeeper.IterateValidators(ctx, func(_ int64, val sdk.ValidatorI) (stop bool) {
		valBz, err := app.StakingKeeper.ValidatorAddressCodec().StringToBytes(val.GetOperator())
		if err != nil {
			panic(err)
		}
		_, _ = app.DistrKeeper.WithdrawValidatorCommission(ctx, valBz)
		return false
	})
	if err != nil {
		panic(err)
	}

	// withdraw all delegator rewards
	dels, err := app.StakingKeeper.GetAllDelegations(ctx)
	if err != nil {
		panic(err)
	}

	for _, delegation := range dels {
		valAddr, err := app.interfaceRegistry.SigningContext().ValidatorAddressCodec().StringToBytes(delegation.ValidatorAddress)
		if err != nil {
			panic(err)
		}

		delAddr, err := app.interfaceRegistry.SigningContext().AddressCodec().StringToBytes(delegation.DelegatorAddress)
		if err != nil {
			panic(err)
		}

		_, _ = app.DistrKeeper.WithdrawDelegationRewards(ctx, delAddr, valAddr)
	}

	// clear validator slash events
	err = app.DistrKeeper.ValidatorSlashEvents.Clear(ctx, nil)
	if err != nil {
		panic(err)
	}

	// clear validator historical rewards
	err = app.DistrKeeper.ValidatorHistoricalRewards.Clear(ctx, nil)
	if err != nil {
		panic(err)
	}

	// set context height to zero
	height := ctx.BlockHeight()
	ctx = ctx.WithBlockHeight(0)

	// reinitialize all validators
	err = app.StakingKeeper.IterateValidators(ctx, func(_ int64, val sdk.ValidatorI) (stop bool) {
		valBz, err := app.StakingKeeper.ValidatorAddressCodec().StringToBytes(val.GetOperator())
		if err != nil {
			panic(err)
		}
		// donate any unwithdrawn outstanding reward tokens to the community pool
		rewards, err := app.DistrKeeper.GetValidatorOutstandingRewardsCoins(ctx, valBz)
		if err != nil {
			panic(err)
		}
		feePool, err := app.DistrKeeper.FeePool.Get(ctx)
		if err != nil {
			panic(err)
		}
		feePool.DecimalPool = feePool.DecimalPool.Add(rewards...) // distribution will allocate this to the protocolpool eventually
		if err := app.DistrKeeper.FeePool.Set(ctx, feePool); err != nil {
			panic(err)
		}

		if err := app.DistrKeeper.Hooks().AfterValidatorCreated(ctx, valBz); err != nil {
			panic(err)
		}
		return false
	})
	if err != nil {
		panic(err)
	}

	// reinitialize all delegations
	for _, del := range dels {
		valAddr, err := app.interfaceRegistry.SigningContext().ValidatorAddressCodec().StringToBytes(del.ValidatorAddress)
		if err != nil {
			panic(err)
		}
		delAddr, err := app.interfaceRegistry.SigningContext().AddressCodec().StringToBytes(del.DelegatorAddress)
		if err != nil {
			panic(err)
		}

		if err := app.DistrKeeper.Hooks().BeforeDelegationCreated(ctx, delAddr, valAddr); err != nil {
			// never called as BeforeDelegationCreated always returns nil
			panic(fmt.Errorf("error while incrementing period: %w", err))
		}

		if err := app.DistrKeeper.Hooks().AfterDelegationModified(ctx, delAddr, valAddr); err != nil {
			// never called as AfterDelegationModified always returns nil
			panic(fmt.Errorf("error while creating a new delegation period record: %w", err))
		}
	}

	// reset context height
	ctx = ctx.WithBlockHeight(height)

	/* Handle staking state. */

	// iterate through redelegations, reset creation height
	err = app.StakingKeeper.IterateRedelegations(ctx, func(_ int64, red stakingtypes.Redelegation) (stop bool) {
		for i := range red.Entries {
			red.Entries[i].CreationHeight = 0
		}
		err = app.StakingKeeper.SetRedelegation(ctx, red)
		if err != nil {
			panic(err)
		}
		return false
	})
	if err != nil {
		panic(err)
	}

	// iterate through unbonding delegations, reset creation height
	err = app.StakingKeeper.UnbondingDelegations.Walk(
		ctx,
		nil,
		func(_ collections.Pair[[]byte, []byte], ubd stakingtypes.UnbondingDelegation) (stop bool, err error) {
			for i := range ubd.Entries {
				ubd.Entries[i].CreationHeight = 0
			}
			err = app.StakingKeeper.SetUnbondingDelegation(ctx, ubd)
			if err != nil {
				return true, err
			}
			return false, err
		},
	)
	if err != nil {
		panic(err)
	}
	// Iterate through validators by power descending, reset bond heights, and
	// update bond intra-tx counters.
	store := ctx.KVStore(app.GetKey(stakingtypes.StoreKey))
	iter := storetypes.KVStoreReversePrefixIterator(store, stakingtypes.ValidatorsKey)
	counter := int16(0)

	for ; iter.Valid(); iter.Next() {
		addr := sdk.ValAddress(stakingtypes.AddressFromValidatorsKey(iter.Key()))
		validator, err := app.StakingKeeper.GetValidator(ctx, addr)
		if err != nil {
			panic("expected validator, not found")
		}

		valAddr, err := app.StakingKeeper.ValidatorAddressCodec().BytesToString(addr)
		if err != nil {
			panic(err)
		}

		validator.UnbondingHeight = 0
		if applyAllowedAddrs && !allowedAddrsMap[valAddr] {
			validator.Jailed = true
		}

		if err = app.StakingKeeper.SetValidator(ctx, validator); err != nil {
			panic(err)
		}
		counter++
	}

	if err := iter.Close(); err != nil {
		app.Logger().Error("error while closing the key-value store reverse prefix iterator: ", err)
		return
	}

	_, err = app.StakingKeeper.ApplyAndReturnValidatorSetUpdates(ctx)
	if err != nil {
		panic(err)
	}
}
