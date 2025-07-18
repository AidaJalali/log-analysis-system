#!/bin/bash
API_URL_BASE="http://localhost:8080/api/projects"
API_KEY=f42cb9fe44310d0ed6913e0b6cb66376
NUM_LOGS=1000
PROJECT_IDS=(f42cb9fe44310d0ed6913e0b6cb66376)

for ((i=1; i<=NUM_LOGS; i++)); do
  PROJECT_ID=${PROJECT_IDS[$RANDOM % ${#PROJECT_IDS[@]}]}
  EVENT_NAME="event_$((RANDOM % 10))"
  TIMESTAMP=$(date +%s)
  
  PAYLOAD1=$((RANDOM % 10000))
  PAYLOAD2=$((RANDOM % 10000))
  PAYLOAD3=$((RANDOM % 10000))

  read -r -d '' JSON_PAYLOAD <<EOF
{
  "event_name": "$EVENT_NAME",
  "timestamp": $TIMESTAMP,
  "payload": {
    "folan1": $PAYLOAD1,
    "folan2": $PAYLOAD2,
    "ye_chi_dige": $PAYLOAD3
  }
}
EOF

  curl -s -X POST "$API_URL_BASE/$PROJECT_ID/logs" \
    -H "Content-Type: application/json" \
    -H "X-API-KEY: $API_KEY" \
    -d "$JSON_PAYLOAD" > /dev/null

  if (( $i % 100 == 0 )); then
    echo "$i logs sent..."
  fi
done

echo "DONE! $NUM_LOGS logs generated and sent."
