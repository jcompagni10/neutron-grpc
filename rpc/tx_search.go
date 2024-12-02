package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	query "github.com/cosmos/cosmos-sdk/types/query"
	"github.com/cosmos/cosmos-sdk/types/tx"
)

func RPCSearch() {
	// Replace with your gRPC server endpoint
	grpcAddress := "https://rpc-lb.neutron.org/"

	// Connect to the gRPC server
	conn, err := grpc.Dial(grpcAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to gRPC server: %v", err)
	}
	defer conn.Close()

	// Create a TxService client
	client := tx.NewServiceClient(conn)

	// Define your search q
	// Example: Search for all transactions sent by a specific sender
	q := "message.module='dex'"

	// Context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Make the tx_search gRPC call
	req := &tx.GetTxsEventRequest{
		Events:  []string{q},
		OrderBy: tx.OrderBy_ORDER_BY_ASC, // Optional: specify order
		// Pagination options (optional)
		Pagination: &query.PageRequest{
			Limit:  10,
			Offset: 0,
		},
	}

	resp, err := client.GetTxsEvent(ctx, req)
	if err != nil {
		log.Fatalf("Failed to query transactions: %v", err)
	}

	// Print the response
	for _, tx := range resp.Txs {
		fmt.Printf("TxHash: %s\n", tx.Body)
		// fmt.Printf("Height: %d\n", tx)
		// fmt.Printf("Gas Used: %d, Gas Wanted: %d\n", tx.GasUsed, tx.GasWanted)
		fmt.Println("----------")
	}
}
