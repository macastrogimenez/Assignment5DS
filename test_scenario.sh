#!/bin/bash

# Test scenario for the distributed auction system
# This script demonstrates the system's functionality and fault tolerance

echo "=== Distributed Auction System Test Scenario ==="
echo ""

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Function to send a bid
send_bid() {
    local bidder=$1
    local amount=$2
    echo -e "${YELLOW}Sending bid: $bidder -> $amount${NC}"
}

# Function to query result
query_result() {
    echo -e "${YELLOW}Querying auction result...${NC}"
}

echo "Test Scenario:"
echo "1. Multiple clients bidding"
echo "2. Bid validation (higher than previous)"
echo "3. Node failure during auction"
echo "4. System continues operating"
echo "5. Query final result"
echo ""

echo "Prerequisites:"
echo "- Start 3 nodes in separate terminals:"
echo "  Terminal 1: make run-node1"
echo "  Terminal 2: make run-node2"
echo "  Terminal 3: make run-node3"
echo ""
echo "Then use the client to test:"
echo ""

echo "=== Example Test Commands ==="
echo ""
echo "# Test 1: Basic bidding"
echo "> bid alice 100"
echo "> bid bob 150"
echo "> bid alice 200"
echo "> result"
echo ""

echo "# Test 2: Invalid bids"
echo "> bid alice 50    # Should fail - lower than previous"
echo "> bid bob 175     # Should fail - lower than highest"
echo "> bid charlie -10 # Should fail - negative amount"
echo ""

echo "# Test 3: Fault tolerance"
echo "1. Place some bids"
echo "2. Kill one node (Ctrl+C)"
echo "3. Continue bidding - should still work"
echo "4. Query result - should get consistent response"
echo ""

echo "# Test 4: Multiple bidders"
echo "> bid alice 100"
echo "> bid bob 120"
echo "> bid charlie 140"
echo "> bid alice 160"
echo "> bid bob 180"
echo "> result"
echo ""

echo "# Test 5: Auction timeout"
echo "Wait 100 seconds or modify AUCTION_DURATION in code"
echo "> bid alice 1000  # Should return EXCEPTION after timeout"
echo "> result          # Should show final winner"
echo ""

echo "=== Expected Behaviors ==="
echo "✓ First bid from a bidder registers them"
echo "✓ Subsequent bids must be higher than bidder's previous bid"
echo "✓ All bids must be higher than current highest bid"
echo "✓ System continues with 1 node failure"
echo "✓ Auction ends after 100 seconds"
echo "✓ Result shows winner and winning bid"
echo ""

echo "=== Running Interactive Test ==="
echo "Start the client with: make run-client"
echo "or: go run client/main.go -nodes=localhost:5001,localhost:5002,localhost:5003"
