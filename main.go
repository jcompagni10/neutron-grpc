package main

import (
	"context"
	"fmt"

	"github.com/jcompagni10/chaincode/rpc"

	"cosmossdk.io/math"
	"google.golang.org/grpc"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/neutron-org/neutron/v5/app/config"
	dextypes "github.com/neutron-org/neutron/v5/x/dex/types"
	"google.golang.org/grpc/metadata"
)

type TrancheUserData struct {
	SharesOwned     math.Int
	SharesWithdrawn math.Int
}

// pion-1:  grpc-falcron.pion-1.ntrn.tech:80
// neutron-1: grpc-kralum.neutron-1.neutron.org:80

func setHeight(ctx context.Context, height int64) context.Context {
	md := metadata.Pairs("x-cosmos-block-height", fmt.Sprintf("%d", height))
	return metadata.NewOutgoingContext(ctx, md)
}
func queryState() error {

	// Create a connection to the gRPC server.
	grpcConn, err := grpc.NewClient(
		"grpc-falcron.pion-1.ntrn.tech:80", // your gRPC server address.
		grpc.WithInsecure(),                // The Cosmos SDK doesn't support any transport security mechanism.
		// This instantiates a general gRPC codec which handles proto bytes. We pass in a nil interface registry
		// if the request/response types contain interface instead of 'nil' you should pass the application specific codec.
		grpc.WithDefaultCallOptions(grpc.ForceCodec(codec.NewProtoCodec(nil).GRPCCodec())),
	)
	if err != nil {
		return err
	}
	defer grpcConn.Close()

	creator := "neutron145sgyzmpe5rj4ssjpk3qksp99tss2k432mmm6h"
	trancheKey := "5cqqun41jhk"
	height := 21403272
	// This creates a gRPC client to query the x/bank service.
	dexClient := dextypes.NewQueryClient(grpcConn)

	ctx := setHeight(context.Background(), int64(height))

	resp, err := dexClient.LimitOrderTrancheUser(ctx, &dextypes.QueryGetLimitOrderTrancheUserRequest{Address: creator, TrancheKey: trancheKey})
	if err != nil {
		panic(err)
	}

	trancheUser := resp.LimitOrderTrancheUser

	trancheResp, err := dexClient.LimitOrderTranche(ctx, &dextypes.QueryGetLimitOrderTrancheRequest{
		PairId:     trancheUser.TradePairId.MustPairID().CanonicalString(),
		TrancheKey: trancheUser.TrancheKey,
		TickIndex:  trancheUser.TickIndexTakerToMaker,
		TokenIn:    trancheUser.TradePairId.MakerDenom,
	})
	if err != nil {
		panic(err)
	}

	fmt.Printf("tranche:%v\n", trancheResp.LimitOrderTranche)

	sharesOwned := resp.LimitOrderTrancheUser.SharesOwned

	withdrawResp, err := dexClient.SimulateCancelLimitOrder(ctx, &dextypes.QuerySimulateCancelLimitOrderRequest{Msg: &dextypes.MsgCancelLimitOrder{Creator: creator, TrancheKey: trancheKey}})

	if err != nil {
		panic(err)
	}

	fmt.Printf("withdraw resp:%v\n", withdrawResp.Resp)

	withdrawValue := dextypes.CalcAmountAsToken0(withdrawResp.Resp.MakerCoinOut.Amount, withdrawResp.Resp.TakerCoinOut.Amount, trancheResp.LimitOrderTranche.MakerPrice)

	fmt.Printf("withdraw val:%v; shares: %v\n", withdrawValue, sharesOwned)

	return nil
}

// func queryState() error {

//	// Create a connection to the gRPC server.
//	grpcConn, err := grpc.Dial(
//		"grpc-kralum.neutron-1.neutron.org:80", // your gRPC server address.
//		grpc.WithInsecure(),                    // The Cosmos SDK doesn't support any transport security mechanism.
//		// This instantiates a general gRPC codec which handles proto bytes. We pass in a nil interface registry
//		// if the request/response types contain interface instead of 'nil' you should pass the application specific codec.
//		grpc.WithDefaultCallOptions(grpc.ForceCodec(codec.NewProtoCodec(nil).GRPCCodec())),
//	)
//	if err != nil {
//		return err
//	}
//	defer grpcConn.Close()

//	// This creates a gRPC client to query the x/bank service.
//	bankclient := banktypes.NewQueryClient(grpcConn)

//	denoms, err := bankclient.TotalSupply(context.Background(), &banktypes.QueryTotalSupplyRequest{Pagination: &query.PageRequest{Limit: 10000, CountTotal: true}})

//	if err != nil {
//		panic(err)
//	}

//	if len(denoms.Supply) != int(denoms.Pagination.Total) {
//		fmt.Printf("missing objects\n")

//	}
//	poolDenoms := make([]sdk.Coin, 0)
//	for _, coin := range denoms.Supply {
//		err := dextypes.ValidatePoolDenom(coin.Denom)

//		if err != nil {
//			continue
//		}
//		poolDenoms = append(poolDenoms, coin)
//		holders, err := bankclient.DenomOwners(context.Background(), &banktypes.QueryDenomOwnersRequest{Denom: coin.Denom})
//		if err != nil {
//			panic(err)
//		}
//		nholder := len(holders.DenomOwners)
//		if nholder > 1 {
//			fmt.Printf("Multi holder*** %v : %v\n", coin.Denom, nholder)
//		} else {
//			fmt.Printf("%v : %v\n", coin.Denom, nholder)
//		}
//	}

//		return nil
//	}
func setup() {
	config.GetDefaultConfig()

}
func main() {
	// setup()
	// err := queryState()
	// if err != nil {
	//	fmt.Printf("err: %v\n", err)

	// }
	rpc.CalcAllVolume()
	// rpc.ExportLOEventData()
}
