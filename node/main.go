package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	pb "auction/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	AUCTION_DURATION = 100 * time.Second
)

type Node struct {
	pb.UnimplementedAuctionServer
	pb.UnimplementedNodeSyncServer

	// Node configuration
	id    int32
	port  string
	peers []string

	// Auction state
	mu            sync.RWMutex
	bids          map[string]int32 // bidder_id -> highest bid amount
	highestBid    int32
	highestBidder string
	startTime     time.Time
	auctionOver   bool

	// gRPC connections to peers
	peerConns map[string]pb.NodeSyncClient
}

func NewNode(id int32, port string, peers []string) *Node {
	return &Node{
		id:            id,
		port:          port,
		peers:         peers,
		bids:          make(map[string]int32),
		highestBid:    0,
		highestBidder: "",
		startTime:     time.Now(),
		auctionOver:   false,
		peerConns:     make(map[string]pb.NodeSyncClient),
	}
}

// Bid handles client bid requests
func (n *Node) Bid(ctx context.Context, req *pb.BidRequest) (*pb.BidResponse, error) {
	n.mu.Lock()
	defer n.mu.Unlock()

	log.Printf("[Node %d] Received bid from %s: %d", n.id, req.BidderId, req.Amount)

	// Check if auction is over
	if n.auctionOver || time.Since(n.startTime) >= AUCTION_DURATION {
		n.auctionOver = true
		return &pb.BidResponse{
			Outcome: pb.BidResponse_EXCEPTION,
			Message: "Auction is over",
		}, nil
	}

	// Validate bid amount
	if req.Amount <= 0 {
		return &pb.BidResponse{
			Outcome: pb.BidResponse_FAIL,
			Message: "Bid amount must be positive",
		}, nil
	}

	// Check if bidder has a previous bid
	previousBid, exists := n.bids[req.BidderId]

	// If bidder exists, new bid must be higher than their previous bid
	if exists && req.Amount <= previousBid {
		return &pb.BidResponse{
			Outcome: pb.BidResponse_FAIL,
			Message: fmt.Sprintf("Bid must be higher than your previous bid of %d", previousBid),
		}, nil
	}

	// Check if bid is higher than current highest
	if req.Amount <= n.highestBid {
		return &pb.BidResponse{
			Outcome: pb.BidResponse_FAIL,
			Message: fmt.Sprintf("Bid must be higher than current highest bid of %d", n.highestBid),
		}, nil
	}

	// Accept the bid
	n.bids[req.BidderId] = req.Amount
	n.highestBid = req.Amount
	n.highestBidder = req.BidderId

	log.Printf("[Node %d] Accepted bid from %s: %d (new highest)", n.id, req.BidderId, req.Amount)

	// Replicate to peers if we're the leader or acting as primary
	go n.replicateBidToPeers(req.BidderId, req.Amount)

	return &pb.BidResponse{
		Outcome: pb.BidResponse_SUCCESS,
		Message: fmt.Sprintf("Bid accepted: %d", req.Amount),
	}, nil
}

// Result handles client result requests
func (n *Node) Result(ctx context.Context, req *pb.ResultRequest) (*pb.ResultResponse, error) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	log.Printf("[Node %d] Result query received", n.id)

	auctionOver := n.auctionOver || time.Since(n.startTime) >= AUCTION_DURATION

	if auctionOver {
		return &pb.ResultResponse{
			AuctionOver: true,
			Winner:      n.highestBidder,
			WinningBid:  n.highestBid,
			Message:     fmt.Sprintf("Auction over. Winner: %s with bid %d", n.highestBidder, n.highestBid),
		}, nil
	}

	return &pb.ResultResponse{
		AuctionOver: false,
		HighestBid:  n.highestBid,
		Message:     fmt.Sprintf("Auction ongoing. Highest bid: %d by %s", n.highestBid, n.highestBidder),
	}, nil
}

// SyncState allows a node to sync state from another node
func (n *Node) SyncState(ctx context.Context, req *pb.SyncStateRequest) (*pb.SyncStateResponse, error) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	log.Printf("[Node %d] State sync requested by node %d", n.id, req.NodeId)

	return &pb.SyncStateResponse{
		Bids:          n.bids,
		HighestBid:    n.highestBid,
		HighestBidder: n.highestBidder,
		StartTime:     n.startTime.Unix(),
	}, nil
}

// ReplicateBid handles bid replication from other nodes
func (n *Node) ReplicateBid(ctx context.Context, req *pb.ReplicateBidRequest) (*pb.ReplicateBidResponse, error) {
	n.mu.Lock()
	defer n.mu.Unlock()

	log.Printf("[Node %d] Replicating bid from %s: %d", n.id, req.BidderId, req.Amount)

	// Check if auction is over
	if n.auctionOver || time.Since(n.startTime) >= AUCTION_DURATION {
		n.auctionOver = true
		return &pb.ReplicateBidResponse{Success: false}, nil
	}

	// Apply the replicated bid
	n.bids[req.BidderId] = req.Amount
	if req.Amount > n.highestBid {
		n.highestBid = req.Amount
		n.highestBidder = req.BidderId
	}

	return &pb.ReplicateBidResponse{Success: true}, nil
}

// replicateBidToPeers sends bid to all peer nodes
func (n *Node) replicateBidToPeers(bidderID string, amount int32) {
	for peerAddr, client := range n.peerConns {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		_, err := client.ReplicateBid(ctx, &pb.ReplicateBidRequest{
			BidderId:  bidderID,
			Amount:    amount,
			Timestamp: time.Now().Unix(),
		})
		cancel()

		if err != nil {
			log.Printf("[Node %d] Failed to replicate to %s: %v", n.id, peerAddr, err)
		} else {
			log.Printf("[Node %d] Successfully replicated to %s", n.id, peerAddr)
		}
	}
}

// connectToPeers establishes gRPC connections to peer nodes
func (n *Node) connectToPeers() {
	for _, peer := range n.peers {
		go func(peerAddr string) {
			// Retry connection with backoff
			for i := 0; i < 10; i++ {
				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
				conn, err := grpc.NewClient(peerAddr,
					grpc.WithTransportCredentials(insecure.NewCredentials()))

				if err != nil {
					cancel()
					log.Printf("[Node %d] Failed to create client for peer %s (attempt %d): %v",
						n.id, peerAddr, i+1, err)
					time.Sleep(time.Duration(i+1) * time.Second)
					continue
				}

				// Test the connection
				syncClient := pb.NewNodeSyncClient(conn)
				_, err = syncClient.SyncState(ctx, &pb.SyncStateRequest{NodeId: n.id})
				cancel()

				if err != nil {
					conn.Close()
					log.Printf("[Node %d] Failed to connect to peer %s (attempt %d): %v",
						n.id, peerAddr, i+1, err)
					time.Sleep(time.Duration(i+1) * time.Second)
					continue
				}

				client := pb.NewNodeSyncClient(conn)
				n.mu.Lock()
				n.peerConns[peerAddr] = client
				n.mu.Unlock()

				log.Printf("[Node %d] Connected to peer %s", n.id, peerAddr)
				return
			}
		}(peer)
	}
}

// checkAuctionTimeout monitors auction duration
func (n *Node) checkAuctionTimeout() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for range ticker.C {
		n.mu.Lock()
		if !n.auctionOver && time.Since(n.startTime) >= AUCTION_DURATION {
			n.auctionOver = true
			log.Printf("[Node %d] Auction ended. Winner: %s with bid %d",
				n.id, n.highestBidder, n.highestBid)
		}
		n.mu.Unlock()
	}
}

func main() {
	nodeID := flag.Int("id", 1, "Node ID")
	port := flag.String("port", "5001", "Port to listen on")
	peersStr := flag.String("peers", "", "Comma-separated list of peer addresses")
	flag.Parse()

	var peers []string
	if *peersStr != "" {
		peers = strings.Split(*peersStr, ",")
	}

	node := NewNode(int32(*nodeID), *port, peers)

	// Start gRPC server
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", *port))
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterAuctionServer(grpcServer, node)
	pb.RegisterNodeSyncServer(grpcServer, node)

	log.Printf("[Node %d] Starting on port %s", node.id, *port)
	log.Printf("[Node %d] Auction duration: %v", node.id, AUCTION_DURATION)
	log.Printf("[Node %d] Peers: %v", node.id, peers)

	// Start background tasks
	go node.connectToPeers()
	time.Sleep(2 * time.Second) // Give peers time to start
	go node.checkAuctionTimeout()

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
