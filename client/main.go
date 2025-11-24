package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	pb "auction/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	nodes       []string
	connections []*grpc.ClientConn
	clients     []pb.AuctionClient
	currentNode int
}

func NewClient(nodes []string) (*Client, error) {
	client := &Client{
		nodes:       nodes,
		connections: make([]*grpc.ClientConn, 0),
		clients:     make([]pb.AuctionClient, 0),
		currentNode: 0,
	}

	// Connect to all nodes
	for i, node := range nodes {
		conn, err := grpc.Dial(node,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithBlock(),
			grpc.WithTimeout(5*time.Second))

		if err != nil {
			log.Printf("Warning: Failed to connect to node %s: %v", node, err)
			continue
		}

		client.connections = append(client.connections, conn)
		client.clients = append(client.clients, pb.NewAuctionClient(conn))
		log.Printf("Connected to node %d: %s", i+1, node)
	}

	if len(client.clients) == 0 {
		return nil, fmt.Errorf("failed to connect to any nodes")
	}

	return client, nil
}

func (c *Client) Close() {
	for _, conn := range c.connections {
		conn.Close()
	}
}

func (c *Client) Bid(bidderID string, amount int32) {
	if len(c.clients) == 0 {
		fmt.Println("Error: No available nodes")
		return
	}

	// Try current node first
	client := c.clients[c.currentNode]
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.Bid(ctx, &pb.BidRequest{
		BidderId: bidderID,
		Amount:   amount,
	})

	if err != nil {
		fmt.Printf("Error communicating with node: %v\n", err)
		// Try next node
		c.currentNode = (c.currentNode + 1) % len(c.clients)
		fmt.Printf("Trying next node...\n")

		ctx2, cancel2 := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel2()

		resp, err = c.clients[c.currentNode].Bid(ctx2, &pb.BidRequest{
			BidderId: bidderID,
			Amount:   amount,
		})

		if err != nil {
			fmt.Printf("Error: Failed to contact any node: %v\n", err)
			return
		}
	}

	switch resp.Outcome {
	case pb.BidResponse_SUCCESS:
		fmt.Printf("Success: %s\n", resp.Message)
	case pb.BidResponse_FAIL:
		fmt.Printf("Fail: %s\n", resp.Message)
	case pb.BidResponse_EXCEPTION:
		fmt.Printf("Exception: %s\n", resp.Message)
	}
}

func (c *Client) Result() {
	if len(c.clients) == 0 {
		fmt.Println("Error: No available nodes")
		return
	}

	client := c.clients[c.currentNode]
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.Result(ctx, &pb.ResultRequest{})

	if err != nil {
		fmt.Printf("Error: %v\n", err)
		// Try next node
		c.currentNode = (c.currentNode + 1) % len(c.clients)

		ctx2, cancel2 := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel2()

		resp, err = c.clients[c.currentNode].Result(ctx2, &pb.ResultRequest{})
		if err != nil {
			fmt.Printf("Error: Failed to contact any node: %v\n", err)
			return
		}
	}

	if resp.AuctionOver {
		fmt.Printf("\n=== AUCTION ENDED ===\n")
		fmt.Printf("Winner: %s\n", resp.Winner)
		fmt.Printf("Winning Bid: %d\n", resp.WinningBid)
		fmt.Printf("====================\n\n")
	} else {
		fmt.Printf("\n=== AUCTION ONGOING ===\n")
		fmt.Printf("Highest Bid: %d\n", resp.HighestBid)
		fmt.Printf("======================\n\n")
	}
}

func (c *Client) InteractiveMode() {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("\n=== Distributed Auction System Client ===")
	fmt.Println("Commands:")
	fmt.Println("  bid <bidder_id> <amount>  - Place a bid")
	fmt.Println("  result                     - Query auction result")
	fmt.Println("  quit                       - Exit")
	fmt.Println("==========================================\n")

	for {
		fmt.Print("> ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		if input == "" {
			continue
		}

		parts := strings.Fields(input)
		command := parts[0]

		switch command {
		case "bid":
			if len(parts) != 3 {
				fmt.Println("Usage: bid <bidder_id> <amount>")
				continue
			}
			bidderID := parts[1]
			amount, err := strconv.Atoi(parts[2])
			if err != nil {
				fmt.Println("Error: amount must be an integer")
				continue
			}
			c.Bid(bidderID, int32(amount))

		case "result":
			c.Result()

		case "quit", "exit":
			fmt.Println("Goodbye!")
			return

		default:
			fmt.Println("Unknown command. Available commands: bid, result, quit")
		}
	}
}

func main() {
	nodesStr := flag.String("nodes", "localhost:5001,localhost:5002,localhost:5003",
		"Comma-separated list of node addresses")
	interactive := flag.Bool("interactive", true, "Run in interactive mode")
	flag.Parse()

	nodes := strings.Split(*nodesStr, ",")

	client, err := NewClient(nodes)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	if *interactive {
		client.InteractiveMode()
	}
}
