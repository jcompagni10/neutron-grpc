package main

import (
	"context"
	"fmt"

	"cosmossdk.io/math"
	"google.golang.org/grpc"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/neutron-org/neutron/v4/app/config"
	math_utils "github.com/neutron-org/neutron/v4/utils/math"
	"github.com/neutron-org/neutron/v4/x/dex/types"
	dextypes "github.com/neutron-org/neutron/v4/x/dex/types"
)

type TrancheUserData struct {
	SharesOwned     math.Int
	SharesWithdrawn math.Int
}

// pion-1:  grpc-falcron.pion-1.ntrn.tech:80
// neutron-1: grpc-kralum.neutron-1.neutron.org:80

func queryState() error {

	// Create a connection to the gRPC server.
	grpcConn, err := grpc.Dial(
		"grpc-kralum.neutron-1.neutron.org:80", // your gRPC server address.
		grpc.WithInsecure(),                    // The Cosmos SDK doesn't support any transport security mechanism.
		// This instantiates a general gRPC codec which handles proto bytes. We pass in a nil interface registry
		// if the request/response types contain interface instead of 'nil' you should pass the application specific codec.
		grpc.WithDefaultCallOptions(grpc.ForceCodec(codec.NewProtoCodec(nil).GRPCCodec())),
	)
	if err != nil {
		return err
	}
	defer grpcConn.Close()

	// This creates a gRPC client to query the x/bank service.
	dexClient := dextypes.NewQueryClient(grpcConn)

	trancheUsers, err := dexClient.LimitOrderTrancheUserAll(context.Background(), &dextypes.QueryAllLimitOrderTrancheUserRequest{Pagination: &query.PageRequest{Limit: 10000, CountTotal: true}})
	if err != nil {
		return err
	}

	if len(trancheUsers.LimitOrderTrancheUser) != int(trancheUsers.Pagination.Total) {
		fmt.Printf("missing objects\n")

	}

	trancheUserData := make(map[string]TrancheUserData)
	tranches := make(map[string]types.LimitOrderTranche)

	for _, tu := range trancheUsers.LimitOrderTrancheUser {
		tk := tu.TrancheKey
		if _, ok := trancheUserData[tk]; !ok {
			req := &dextypes.QueryGetLimitOrderTrancheRequest{
				PairId:     tu.TradePairId.MustPairID().CanonicalString(),
				TokenIn:    tu.TradePairId.MakerDenom,
				TickIndex:  tu.TickIndexTakerToMaker,
				TrancheKey: tk,
			}
			tranche, err := dexClient.LimitOrderTranche(context.Background(), req)
			if err != nil {
				fmt.Printf("failed to get tranche :%v \n tu: %v \n", req, tu)
				continue
			}
			trancheUserData[tk] = TrancheUserData{SharesOwned: tu.SharesOwned, SharesWithdrawn: tu.SharesWithdrawn}
			tranches[tk] = *tranche.LimitOrderTranche
		} else {
			userData := trancheUserData[tk]
			userData.SharesOwned = userData.SharesOwned.Add(tu.SharesOwned)
			userData.SharesWithdrawn = userData.SharesWithdrawn.Add(tu.SharesWithdrawn)
			trancheUserData[tk] = userData
		}

	}

	for trancheKey, v := range trancheUserData {
		tranche, ok := tranches[trancheKey]
		if !ok {
			fmt.Printf("tranche not found: %v\n", trancheKey)

		}

		if !tranche.TotalMakerDenom.Equal(v.SharesOwned) {
			fmt.Printf("val share mismatch. Tranche: %v, shares: %v\n", tranche.TotalMakerDenom, v.SharesOwned)

		}

		sharesWithdrawnTaker := math_utils.NewPrecDecFromInt(v.SharesWithdrawn).Quo(tranche.PriceTakerToMaker)
		expectedTaker := sharesWithdrawnTaker.TruncateInt().Add(tranche.ReservesTakerDenom)

		if !tranche.TotalTakerDenom.Equal(expectedTaker) {
			fmt.Printf("totalTaker mismatch. Tranche: %v, expected: %v\n tk: %s", tranche.TotalMakerDenom, expectedTaker, trancheKey)

		}

		fmt.Printf("%s : sharesOwned: %v, shares withdrawn: %v; expectedTaked: %v; totalTaker: %v\n", trancheKey, v.SharesOwned, v.SharesWithdrawn, expectedTaker, tranche.TotalTakerDenom)

	}

	return nil
}

func setup() {
	config.GetDefaultConfig()

}
func main() {
	setup()
	err := queryState()
	if err != nil {
		fmt.Printf("err: %v\n", err)

	}
}
