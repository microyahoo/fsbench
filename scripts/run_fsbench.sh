#!/bin/bash

# Define variables
SERVER_URL="http://s3-smd.deeproute.cn/smd-pkg/tools/fsbench/fsbench-server"
WORKER_URL="http://s3-smd.deeproute.cn/smd-pkg/tools/fsbench/fsbench-worker"
CONFIG_URL="http://s3-smd.deeproute.cn/smd-pkg/tools/fsbench/fsbench-config.yaml"

SERVER_IP="10.3.8.1"
# Define: mixed continuous ranges and discrete IPs
WORKER_RANGES=(
    "10.3.8.1-30"    # Continuous range: 1,2,...,30
    "10.3.8.237-239"    # Continuous range: 237,238,239
)

# Generate all worker IP addresses
WORKER_IPS=()
for range in "${WORKER_RANGES[@]}"; do
    if [[ "$range" =~ ^([0-9]+\.[0-9]+\.[0-9]+\.)([0-9]+)-([0-9]+)$ ]]; then
        # Process continuous range
        prefix="${BASH_REMATCH[1]}"
        start="${BASH_REMATCH[2]}"
        end="${BASH_REMATCH[3]}"
        for ((i=start; i<=end; i++)); do
            WORKER_IPS+=("${prefix}${i}")
        done
    else
        # Single IP
        WORKER_IPS+=("$range")
    fi
done

SERVER_PORT="2000"

# 1. Check if fsbench-config.yaml exists
if [ ! -f "fsbench-config.yaml" ]; then
    echo "fsbench-config.yaml not found, downloading..."
    if ! wget -q "$CONFIG_URL" -O fsbench-config.yaml
    then
        echo "Error: Failed to download fsbench-config.yaml"
        exit 1
    fi
    echo "fsbench-config.yaml downloaded successfully. Please check and modify it if needed, then run this script again."
    exit 0
else
    echo "Using local fsbench-config.yaml"
fi

# 2. Download server and worker binaries
echo "Downloading fsbench files..."
wget -q "$SERVER_URL" -O fsbench-server
wget -q "$WORKER_URL" -O fsbench-worker

# Verify download success
if [ ! -f "fsbench-server" ] || [ ! -f "fsbench-worker" ]; then
    echo "Error: Failed to download required files"
    exit 1
fi

# Make binaries executable
chmod +x fsbench-server fsbench-worker

# 3. Start server on the server node
echo "Starting fsbench server on $SERVER_IP..."
nohup ./fsbench-server --config.file fsbench-config.yaml > server.log 2>&1 &

# Wait for server to initialize
sleep 5

# 4. Start workers on all nodes (including server node)
for ip in "${WORKER_IPS[@]}"; do
    echo "Starting fsbench worker on $ip..."
    if [ "$ip" == "$SERVER_IP" ]; then
        # Local execution
        nohup ./fsbench-worker --server.address "$SERVER_IP:$SERVER_PORT" > "worker-$ip.log" 2>&1 &
    else
        # Remote execution via SSH
        ssh -o ConnectTimeout=30 -f root@$ip "nohup bash -c 'wget -q $WORKER_URL -O fsbench-worker && chmod +x fsbench-worker && nohup ./fsbench-worker --server.address $SERVER_IP:$SERVER_PORT > worker-$ip.log 2>&1' &"
    fi
done

echo "All fsbench workers started."
echo "Server is running on $SERVER_IP, workers are running on all nodes."
echo "Server logs: server.log"
echo "Worker logs: worker-<ip>.log"
