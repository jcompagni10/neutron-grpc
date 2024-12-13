package rpc

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"encoding/gob"

	"encoding/csv"

	abcitypes "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx"
)

func CalcAllVolume() {
	// Replace with your gRPC server endpoint
	grpcAddress := "grpc-kralum.neutron-1.neutron.org:80"

	// Connect to the gRPC server
	conn, err := grpc.Dial(grpcAddress,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(16*1024*1024)),
	)
	if err != nil {
		log.Fatalf("Failed to connect to gRPC server: %v", err)
	}
	defer conn.Close()

	// Create a TxService client
	client := tx.NewServiceClient(conn)

	minBlock := 17707754
	maxBlock := 17753559
	step := 5_000

	var allTxResps []*sdk.TxResponse

	for queryStartBlock := minBlock; queryStartBlock <= maxBlock; queryStartBlock += step {

		queryEndBlock := queryStartBlock + step
		fmt.Printf("outer fetch min: %d; max: %v\n", queryStartBlock, queryEndBlock)
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

	var loEvents []abcitypes.Event

	if len(allTxResps) > 0 {
		fmt.Printf("got %d resps\n", len(allTxResps))

		for _, resp := range allTxResps {
			events := filterEvents(resp.Events, "message", "action", "PlaceLimitOrder")
			fmt.Printf("got n event:%v\n", len(events))

			loEvents = append(loEvents, events...)
		}

		fmt.Printf("n LO events:%v\n", len(loEvents))

	} else {
		log.Panic("no txs found")
	}

	writeEventsToFile("lo_events.gob", loEvents)
}

func writeEventsToFile(filename string, data []abcitypes.Event) {

	file, err := os.Create(filename)
	if err != nil {
		fmt.Printf("Failed to create file: %v\n", err)
		return
	}
	defer file.Close()

	encoder := gob.NewEncoder(file)
	err = encoder.Encode(data)
	if err != nil {
		fmt.Printf("Failed to encode struct to Gob: %v\n", err)
		return
	}

	fmt.Println("wrote events to file")

}

func readEventsFromFile(filename string) []abcitypes.Event {

	file, err := os.Open(filename)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	var data []abcitypes.Event
	decoder := gob.NewDecoder(file)
	err = decoder.Decode(&data)
	if err != nil {
		panic(err)
	}

	return data
}

func ExportLOEventData() {

	rawData := readEventsFromFile("lo_events.gob")

	var data []LOData

	for _, evt := range rawData {
		data = append(data, eventToLOData(evt))
	}

	file, err := os.Create("output.csv")
	if err != nil {
		fmt.Printf("Failed to create file: %v\n", err)
		return
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush() // Ensure all data is written to the file

	// Write the header row
	header := []string{
		"Creator",
		"TokenIn",
		"TokenOut",
		"AmountIn",
		"OrderType",
		"SwapAmountIn",
		"SwapAmountOut",
	}
	if err := writer.Write(header); err != nil {
		fmt.Printf("Failed to write header: %v\n", err)
		return
	}

	// Write the struct data
	for _, evt := range data {
		row := []string{
			evt.Creator,
			evt.TokenIn,
			evt.TokenOut,
			strconv.Itoa(int(evt.AmountIn)),
			evt.OrderType,
			strconv.Itoa(int(evt.SwapAmountIn)),
			strconv.Itoa(int(evt.SwapAmountOut)),
		}
		if err := writer.Write(row); err != nil {
			fmt.Printf("Failed to write row: %v\n", err)
			return
		}
	}

	fmt.Println("Data successfully written to output.csv!")

}

type LOData struct {
	Creator       string
	TokenIn       string
	TokenOut      string
	AmountIn      int64
	OrderType     string
	SwapAmountIn  int64
	SwapAmountOut int64
	Tick          int64
}

func eventToLOData(event abcitypes.Event) LOData {
	amountIn, err := strconv.ParseInt(getEventKey(event, "AmountIn"), 10, 0)
	if err != nil {
		amountIn = 0
	}

	swapAmountIn, err := strconv.ParseInt(getEventKey(event, "SwapAmountIn"), 10, 0)
	if err != nil {
		swapAmountIn = 0
	}

	swapAmountOut, err := strconv.ParseInt(getEventKey(event, "SwapAmountOut"), 10, 0)
	if err != nil {
		swapAmountOut = 0
	}
	tick, err := strconv.ParseInt(getEventKey(event, "LimitTick"), 10, 0)
	if err != nil {
		tick = 0
	}

	return LOData{
		Creator:       getEventKey(event, "Creator"),
		TokenIn:       getEventKey(event, "TokenIn"),
		TokenOut:      getEventKey(event, "TokenOut"),
		AmountIn:      amountIn,
		OrderType:     getEventKey(event, "OrderType"),
		SwapAmountIn:  swapAmountIn,
		SwapAmountOut: swapAmountOut,
		Tick:          tick,
	}

}

func getEventKey(event abcitypes.Event, key string) string {
	for _, attr := range event.Attributes {

		if attr.Key == key {
			return attr.Value
		}
	}

	return ""

}

func getTxEvents(client tx.ServiceClient, query string) ([]*tx.GetTxsEventResponse, error) {
	var allResps []*tx.GetTxsEventResponse

	pageLimit := 100
	for page := 1; ; page++ {
		fmt.Printf("fetching page: %d\n", page)

		// Make the tx_search gRPC call
		req := &tx.GetTxsEventRequest{
			Query:   query,
			OrderBy: tx.OrderBy_ORDER_BY_ASC, // Optional: specify order
			// Pagination options (optional)
			Page:  uint64(page),
			Limit: uint64(pageLimit),
		}

		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		resp, err := client.GetTxsEvent(ctx, req)

		if err != nil {
			// // Pagination logic is broken, this is fine. We are out of responses
			// match, _ := regexp.MatchString("page should be within", err.Error())
			// if match {
			//	fmt.Printf("total got:%v\n", len(allResps))

			//	return allResps, nil
			// }
			// return []*tx.GetTxsEventResponse{}, err
			panic(err)
		}

		fmt.Printf(" total:%v; n txs: %v; n tx responses: %v\n", resp.Total, len(resp.Txs), len(resp.TxResponses))

		if len(resp.Txs) > 0 {
			allResps = append(allResps, resp)
		} else {
			break
		}

		if resp.Total <= uint64(page*pageLimit) {
			break
		}

	}

	return allResps, nil

}

func filterEvents(events []abcitypes.Event, evtType, key, value string) []abcitypes.Event {
	var keptEvents []abcitypes.Event

	for _, evt := range events {
		if evt.Type != evtType {
			continue
		}
		if hasKVPair(evt, key, value) {
			keptEvents = append(keptEvents, evt)
		}
	}
	return keptEvents
}

func hasKVPair(event abcitypes.Event, key, value string) bool {
	for _, attr := range event.Attributes {
		if attr.Key == key {
			if attr.Value == value {
				return true
			} else {
				return false
			}
		}

	}
	return false
}
