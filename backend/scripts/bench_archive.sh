#!/bin/bash

# Apache Bench (ab) Load Test Scripts for Search Events API
# Note: ab has significant limitations compared to k6:
# - Can only test one URL at a time
# - No ramping support (must run multiple commands sequentially)
# - Basic auth must be passed differently
# - No dynamic scenario selection
# - No custom metrics or checks beyond basic response stats

# Configuration
API_URL="http://localhost:8080"  # Replace with your actual API URL

# User credentials (base64 encoded for Basic Auth)
# Format: email:password encoded in base64
USER1_AUTH="YXlzdWx0YW5fdGFsZ2F0XzFAZmVzdC50aXg6LzhlQyRBRD4="  # aysultan_talgat_1@fest.tix:/8eC$AD>
USER2_AUTH="YXlhdWx5bV9iYXphcmJhZXZhXzNAcXVpY2sucGFzczpMRGI2MF8lXTQ="  # ayaulym_bazarbaeva_3@quick.pass:LDb60_%]4
USER3_AUTH="c3VsdGFuX3N1bHRhbm92XzJAc2hvdy5nbzoqSVZTZj9raCl4YQ=="  # sultan_sultanov_2@show.go:*IVSf?kh)xa

# Search phrases (URL encoded)
SEARCH_PHRASES=(
    "концерт"
    "Дмитрий%20Козлов"
    "азарт"
    "Чемпионат"
    "Красная%20шапочка"
    "Экспозиция%20%22Звездные%20войны%22"
)

# Function to run ab test with specific parameters
run_ab_test() {
    local url=$1
    local concurrency=$2
    local requests=$3
    local auth=$4
    local test_name=$5
    
    echo "========================================"
    echo "Running test: $test_name"
    echo "URL: $url"
    echo "Concurrency: $concurrency"
    echo "Total requests: $requests"
    echo "========================================"
    
    ab -n $requests \
       -c $concurrency \
       -H "Authorization: Basic $auth" \
       -H "Accept: application/json" \
       -g "${test_name}_results.tsv" \
       "$url"
    
    echo ""
}

# Function to simulate ramping (manual approximation)
simulate_ramping() {
    local base_url=$1
    local auth=$2
    local test_prefix=$3
    
    echo "Starting ramping simulation for: $test_prefix"
    
    # Phase 1: Ramp up (2 minutes) - gradually increase from 0 to 1000 req/s
    echo "Phase 1: Ramping up (2 minutes)"
    
    # Start with low concurrency and increase
    run_ab_test "$base_url" 50 2000 "$auth" "${test_prefix}_ramp_1"
    sleep 10
    
    run_ab_test "$base_url" 100 5000 "$auth" "${test_prefix}_ramp_2"
    sleep 10
    
    run_ab_test "$base_url" 200 10000 "$auth" "${test_prefix}_ramp_3"
    sleep 10
    
    run_ab_test "$base_url" 500 30000 "$auth" "${test_prefix}_ramp_4"
    sleep 10
    
    run_ab_test "$base_url" 1000 60000 "$auth" "${test_prefix}_ramp_5"
    
    # Phase 2: Sustained load (5 minutes) - maintain 1000 req/s
    echo "Phase 2: Sustained load (5 minutes)"
    run_ab_test "$base_url" 1000 300000 "$auth" "${test_prefix}_sustained"
    
    # Phase 3: Ramp down (1 minute) - decrease from 1000 to 0 req/s
    echo "Phase 3: Ramping down (1 minute)"
    run_ab_test "$base_url" 500 30000 "$auth" "${test_prefix}_ramp_down_1"
    sleep 5
    run_ab_test "$base_url" 200 10000 "$auth" "${test_prefix}_ramp_down_2"
    sleep 5
    run_ab_test "$base_url" 50 2000 "$auth" "${test_prefix}_ramp_down_3"
}

# Main test scenarios
echo "========================================="
echo "Search Events Load Test - Apache Bench"
echo "========================================="
echo ""

# Test Scenario 1: Simple query search (fixed page=1, pageSize=20)
echo "SCENARIO 1: Simple Query Search"
for phrase in "${SEARCH_PHRASES[@]}"; do
    URL="${API_URL}/api/events?query=${phrase}&page=1&pageSize=20"
    
    # Quick test with each phrase
    run_ab_test "$URL" 100 1000 "$USER1_AUTH" "query_search_${phrase}"
done

echo ""
echo "SCENARIO 2: Query with Fixed Date"
# Test Scenario 2: Query with fixed date (fixed page=1, pageSize=20)
for phrase in "${SEARCH_PHRASES[@]}"; do
    URL="${API_URL}/api/events?query=${phrase}&date=2024-12-25&page=1&pageSize=20"
    
    # Quick test with each phrase and date
    run_ab_test "$URL" 100 1000 "$USER2_AUTH" "date_search_${phrase}"
done

echo ""
echo "SCENARIO 3: Full Load Test Simulation"
# Pick one URL for the full ramping simulation (most representative)
MAIN_TEST_URL="${API_URL}/api/events?query=концерт&page=1&pageSize=20"
simulate_ramping "$MAIN_TEST_URL" "$USER3_AUTH" "main_load_test"

echo ""
echo "========================================="
echo "Load Test Complete!"
echo "========================================="
echo "Results saved in *_results.tsv files"
echo ""
echo "To analyze results:"
echo "  - Check Apache Bench output for response times and throughput"
echo "  - Use gnuplot or similar tools to graph the .tsv files"
echo "  - Look for:"
echo "    * Requests per second"
echo "    * Time per request (mean)"
echo "    * Failed requests"
echo "    * 95th and 99th percentile response times"

# Alternative: Single command examples for manual testing
echo ""
echo "========================================="
echo "Single Command Examples (for manual testing):"
echo "========================================="
echo ""
echo "# Basic test with 1000 requests, 100 concurrent:"
echo "ab -n 1000 -c 100 -H \"Authorization: Basic $USER1_AUTH\" -H \"Accept: application/json\" \"${API_URL}/api/events?query=концерт&page=1&pageSize=20\""
echo ""
echo "# High load test with 10000 requests, 500 concurrent:"
echo "ab -n 10000 -c 500 -H \"Authorization: Basic $USER1_AUTH\" -H \"Accept: application/json\" \"${API_URL}/api/events?query=концерт&page=1&pageSize=20\""
echo ""
echo "# Sustained load test with 100000 requests, 1000 concurrent:"
echo "ab -n 100000 -c 1000 -H \"Authorization: Basic $USER1_AUTH\" -H \"Accept: application/json\" \"${API_URL}/api/events?query=концерт&page=1&pageSize=20\""
echo ""
echo "# With keep-alive connections:"
echo "ab -k -n 10000 -c 500 -H \"Authorization: Basic $USER1_AUTH\" -H \"Accept: application/json\" \"${API_URL}/api/events?query=концерт&page=1&pageSize=20\""
echo ""
echo "# With timeout of 30 seconds (matching k6 script):"
echo "ab -s 30 -n 10000 -c 500 -H \"Authorization: Basic $USER1_AUTH\" -H \"Accept: application/json\" \"${API_URL}/api/events?query=концерт&page=1&pageSize=20\""

# Helper script to parse ab output
cat << 'EOF' > parse_ab_results.sh
#!/bin/bash
# Helper script to extract key metrics from ab output

if [ "$1" == "" ]; then
    echo "Usage: $0 <ab_output_file>"
    exit 1
fi

echo "Key Metrics from AB Test:"
echo "========================="
grep "Requests per second" $1
grep "Time per request" $1 | head -1
grep "Failed requests" $1
grep "Total transferred" $1
grep "HTML transferred" $1
grep "50%" $1
grep "90%" $1
grep "95%" $1
grep "99%" $1
EOF

chmod +x parse_ab_results.sh

echo ""
echo "Helper script 'parse_ab_results.sh' created to extract key metrics."
echo "Usage: ./parse_ab_results.sh <ab_output_file>"