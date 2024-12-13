package rpc

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"testing"

	abcitypes "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx"
	dextypes "github.com/neutron-org/neutron/v5/x/dex/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

const sender = "neutron1nz852flh6np9xlg9ju3ka6w5txezsxt0j4lypn"

var Conn *grpc.ClientConn

func setHeight(ctx context.Context, height int64) context.Context {
	md := metadata.Pairs("x-cosmos-block-height", fmt.Sprintf("%d", height))
	return metadata.NewOutgoingContext(ctx, md)
}

var seenTranches = make(map[string]dextypes.LimitOrderTranche)

func queryLimitOrderTranche(creator string, trancheKey string, height int64) dextypes.LimitOrderTranche {
	val, ok := seenTranches[trancheKey]
	if ok {
		return val
	}
	dexClient := dextypes.NewQueryClient(Conn)
	ctx := setHeight(context.Background(), height)

	resp, err := dexClient.LimitOrderTrancheUser(ctx, &dextypes.QueryGetLimitOrderTrancheUserRequest{Address: creator, TrancheKey: trancheKey, CalcWithdrawableShares: false})
	if err != nil {
		return dextypes.LimitOrderTranche{}
	}

	trancheUser := resp.LimitOrderTrancheUser

	trancheResp, err := dexClient.LimitOrderTranche(ctx, &dextypes.QueryGetLimitOrderTrancheRequest{
		PairId:     trancheUser.TradePairId.MustPairID().CanonicalString(),
		TrancheKey: trancheUser.TrancheKey,
		TickIndex:  trancheUser.TickIndexTakerToMaker,
		TokenIn:    trancheUser.TradePairId.MakerDenom,
	})
	if err != nil {
		return dextypes.LimitOrderTranche{}
	}
	tranche := *trancheResp.LimitOrderTranche
	seenTranches[trancheKey] = tranche

	return tranche

}

func TxDebug() {
	// Replace with your gRPC server endpoint
	grpcAddress := "grpc-kralum.neutron-1.neutron.org:80"

	// Connect to the gRPC server
	var err error
	Conn, err = grpc.Dial(grpcAddress,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(16*1024*1024)),
		grpc.WithDefaultCallOptions(grpc.ForceCodec(codec.NewProtoCodec(nil).GRPCCodec())),
	)
	if err != nil {
		log.Fatalf("Failed to connect to gRPC server: %v", err)
	}
	defer Conn.Close()

	// Create a TxService client
	client := tx.NewServiceClient(Conn)

	minBlock := 17701062
	maxBlock := 17701403
	step := 5000

	var allTxResps []*sdk.TxResponse

	for queryStartBlock := minBlock; queryStartBlock <= maxBlock; queryStartBlock += step {

		queryEndBlock := min(queryStartBlock+step, maxBlock)

		query := fmt.Sprintf("tx.height>%d AND tx.height<=%d  AND message.module='dex'",
			queryStartBlock,
			queryEndBlock,
		)
		resps, err := getTxEvents(client, query)
		if err != nil {
			panic(err)
		}

		for _, resp := range resps {
			allTxResps = append(allTxResps, resp.TxResponses...)
		}

	}

	fmt.Printf("got n:%v\n", len(allTxResps))

	var allEventData []EventData

	if len(allTxResps) > 0 {
		for _, resp := range allTxResps {
			events := filterEvents(resp.Events, "message", "TokenOne", "untrn")
			for _, evt := range events {
				prettyEvent := eventToData(evt, resp.Height)

				allEventData = append(allEventData, prettyEvent)
				fmt.Printf("%#v\n", prettyEvent)

			}

		}

	} else {
		log.Panic("no txs found")
	}

	fmt.Printf("got %d events\n", len(allEventData))

}

type EventData struct {
	Action          string
	TokenIn         string
	TokenOut        string
	AmountIn        int64
	CancelAmountOut string
	SwapAmountOut   string
	SwapAmountIn    string
	Creator         string
	TickIndex       int64
	Height          int64
}

func eventToData(event abcitypes.Event, height int64) EventData {
	amountIn, err := strconv.ParseInt(getEventKey(event, "AmountIn"), 10, 0)
	if err != nil {
		amountIn = 0
	}

	cancelAmountOut := getEventKey(event, "AmountOut")

	SwapAmountOut := getEventKey(event, "SwapAmountOut")
	SwapAmountIn := getEventKey(event, "SwapAmountIn")

	action := getEventKey(event, "action")

	tickIndex, err := strconv.ParseInt(getEventKey(event, "LimitTick"), 10, 0)

	if err != nil {

		trancheKey := getEventKey(event, "TrancheKey")

		queryHeight := height

		if action == "CancelLimitOrder" || action == "WithdrawLimitOrder" {
			queryHeight--
		}
		tranche := queryLimitOrderTranche("neutron13tyej6xvgj4uc4c2xxgktjkgllzgngaksqkvyv42lx6y82teum6q3307xw", trancheKey, queryHeight)
		if tranche.Key != nil {
			tickIndex = tranche.Key.TickIndexTakerToMaker
		}

	} else {

		// Reverse LimitTick to get trancheTick
		tickIndex = tickIndex * -1

	}

	return EventData{
		Creator:         getEventKey(event, "Creator"),
		TokenIn:         getEventKey(event, "TokenIn"),
		TokenOut:        getEventKey(event, "TokenOut"),
		AmountIn:        amountIn,
		Action:          action,
		CancelAmountOut: cancelAmountOut,
		SwapAmountOut:   SwapAmountOut,
		SwapAmountIn:    SwapAmountIn,
		TickIndex:       tickIndex,
		Height:          height,
	}

}

func TestEvent(t *testing.T) {

}
