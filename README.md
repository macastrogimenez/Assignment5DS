# Distributed Auction System

A distributed auction system implemented in Go using gRPC with replication to handle crash failures.

## Features

- **Distributed Architecture**: Multiple nodes running on separate processes
- **Fault Tolerance**: Resilient to at least one (1) node crash failure
- **gRPC Communication**: Reliable, ordered message transport
- **Bid Validation**: Ensures bids are higher than previous bids
- **Time-based Auction**: Auction ends after 100 seconds
- **State Replication**: All nodes maintain consistent auction state

## Architecture

The system consists of:

1. **Nodes (Servers)**: Handle auction logic and maintain replicated state
2. **Clients**: Connect to any available node to place bids or query results
3. **Internal Replication**: Nodes replicate bid state to ensure consistency

### Node Components

Each node maintains:
- Auction state (bids, highest bidder, auction timer)
- Peer connections for replication
- Leader election mechanism
- Heartbeat system for failure detection

## Prerequisites

- Go 1.21 or higher
- Protocol Buffers compiler (protoc)
- Make (optional, for using Makefile)

## Installation

1. Install Protocol Buffers compiler:
```bash
# macOS
brew install protobuf

# Install Go plugins for protoc
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

2. Generate protobuf files and install dependencies:
```bash
make proto
go mod tidy
```

## Running the System

### Option 1: Using Makefile (Recommended)

**Terminal 1 - Start Node 1:**
```bash
make run-node1
```

**Terminal 2 - Start Node 2:**
```bash
make run-node2
```

**Terminal 3 - Start Node 3:**
```bash
make run-node3
```

**Terminal 4 - Start Client:**
```bash
make run-client
```

### Option 2: Manual Execution

**Start nodes manually:**
```bash
# Terminal 1
go run node/main.go -id=1 -port=5001 -peers=localhost:5002,localhost:5003

# Terminal 2
go run node/main.go -id=2 -port=5002 -peers=localhost:5001,localhost:5003

# Terminal 3
go run node/main.go -id=3 -port=5003 -peers=localhost:5001,localhost:5002
```

**Start client:**
```bash
go run client/main.go -nodes=localhost:5001,localhost:5002,localhost:5003
```

## Using the Client

The client provides an interactive command-line interface:

```
> bid alice 100
✓ Success: Bid accepted: 100

> bid bob 150
✓ Success: Bid accepted: 150

> bid alice 120
✗ Fail: Bid must be higher than your previous bid of 150

> result
=== AUCTION ONGOING ===
Highest Bid: 150
======================

> quit
```

### Commands

- `bid <bidder_id> <amount>` - Place a bid
- `result` - Query the current auction state
- `quit` - Exit the client

## API Specification

### Client-to-Node API

#### Bid
- **Method**: `Bid`
- **Input**: `amount` (int32), `bidder_id` (string)
- **Output**: `outcome` (SUCCESS, FAIL, or EXCEPTION)
- **Behavior**:
  - First call registers the bidder
  - Subsequent bids must be higher than bidder's previous bid
  - Bid must be higher than current highest bid
  - Returns EXCEPTION if auction is over

#### Result
- **Method**: `Result`
- **Input**: void
- **Output**: `outcome` (auction state)
- **Behavior**:
  - If auction is over: returns winner and winning bid
  - If auction is ongoing: returns current highest bid

## Testing Fault Tolerance

To test the system's resilience to node failures:

1. Start all three nodes and the client
2. Place some bids through the client
3. **Kill one node** (Ctrl+C in that node's terminal)
4. Continue placing bids - the system should still work
5. Query results - should still get consistent responses

Example:
```bash
# Start all nodes and client
# In client:
> bid alice 100
> bid bob 200

# Kill node 2 (Ctrl+C in terminal 2)

# Continue bidding:
> bid charlie 300
# Should still work!

> result
# Should show consistent state
```

## System Properties

### Consistency Model

The system implements **Primary-Backup Replication** with the following properties:

- **Write Operations**: Bids are accepted by any node and replicated to peers
- **Read Operations**: Results can be queried from any available node
- **Failure Handling**: If one node fails, other nodes continue serving requests
- **State Synchronization**: All nodes maintain eventually consistent state

### Auction Rules

1. First bid from a bidder registers them
2. Each subsequent bid must be higher than bidder's previous bid
3. All bids must be higher than current highest bid
4. Auction runs for 100 seconds from system start
5. After timeout, no new bids accepted
6. Highest bidder wins the auction

## Project Structure

```
Assignment5/
├── go.mod                 # Go module definition
├── Makefile              # Build and run commands
├── README.md             # This file
├── proto/
│   └── auction.proto     # Protocol buffer definitions
├── node/
│   └── main.go          # Node server implementation
├── client/
│   └── main.go          # Client implementation
└── test_scenario.sh     # Test script (optional)
```

## Implementation Details

### Replication Strategy

- **Primary-Backup**: Any node can accept writes and replicate to others
- **Heartbeats**: Nodes send periodic heartbeats to detect failures
- **State Sync**: Nodes can sync state from peers on startup or recovery

### Fault Tolerance

- System tolerates **1 node failure** out of 3 nodes
- Clients automatically failover to available nodes
- State is replicated across all nodes

### Time Constraints

- Auction duration: 100 seconds
- Heartbeat interval: 2 seconds
- Leader timeout: 5 seconds
- RPC timeout: 2-5 seconds

## Troubleshooting

**Error: "Failed to connect to peer"**
- Ensure all nodes are started
- Check port availability
- Wait a few seconds for connections to establish

**Error: "No available nodes"**
- At least one node must be running
- Check network connectivity
- Verify node addresses in client command

**Error: "expected ';', found 'EOF'"**
- Run `make proto` to generate protobuf files
- Run `go mod tidy` to download dependencies

## Building for Production

```bash
# Generate protobuf files
make proto

# Build binaries
go build -o bin/node ./node
go build -o bin/client ./client

# Run
./bin/node -id=1 -port=5001 -peers=...
./bin/client -nodes=...
```
