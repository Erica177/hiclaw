#!/bin/bash
# create-worker.sh - Worker creation via hiclaw-controller API
#
# Thin wrapper that delegates all provisioning (Matrix, Gateway, MinIO,
# container startup) to the hiclaw-controller HTTP API.
#
# Usage:
#   create-worker.sh --name <NAME> [--model <MODEL_ID>] [--image <IMAGE>]
#     [--runtime openclaw|copaw] [--skills s1,s2] [--mcp-servers s1,s2]
#     [--role worker|team_leader] [--team <TEAM>] [--team-leader <LEADER>]
#
# Prerequisites:
#   - SOUL.md must already exist at /root/hiclaw-fs/agents/<NAME>/SOUL.md
#   - Environment: HICLAW_CONTROLLER_URL (or defaults to http://localhost:8090),
#     HICLAW_CONTROLLER_API_KEY

set -e

# ============================================================
# Logging
# ============================================================
log() {
    local msg="[hiclaw $(date '+%Y-%m-%d %H:%M:%S')] $1"
    echo "${msg}"
    if [ -w /proc/1/fd/1 ]; then
        echo "${msg}" > /proc/1/fd/1
    fi
}

# ============================================================
# Parse arguments
# ============================================================
WORKER_NAME=""
MODEL_ID=""
WORKER_RUNTIME=""
CUSTOM_IMAGE=""
WORKER_SKILLS=""
MCP_SERVERS=""
WORKER_ROLE=""
TEAM_NAME=""
TEAM_LEADER_NAME=""
SOUL_CONTENT=""

while [ $# -gt 0 ]; do
    case "$1" in
        --name)        WORKER_NAME="$2"; shift 2 ;;
        --model)       MODEL_ID="$2"; shift 2 ;;
        --runtime)     WORKER_RUNTIME="$2"; shift 2 ;;
        --image)       CUSTOM_IMAGE="$2"; shift 2 ;;
        --skills)      WORKER_SKILLS="$2"; shift 2 ;;
        --mcp-servers) MCP_SERVERS="$2"; shift 2 ;;
        --role)        WORKER_ROLE="$2"; shift 2 ;;
        --team)        TEAM_NAME="$2"; shift 2 ;;
        --team-leader) TEAM_LEADER_NAME="$2"; shift 2 ;;
        *) echo "Unknown option: $1"; exit 1 ;;
    esac
done

if [ -z "${WORKER_NAME}" ]; then
    echo "Usage: create-worker.sh --name <NAME> [--model <MODEL_ID>] [--image <IMAGE>] [--runtime openclaw|copaw] [--skills s1,s2] [--mcp-servers s1,s2] [--role worker|team_leader] [--team <TEAM>] [--team-leader <LEADER>]"
    exit 1
fi

# Normalize to lowercase (Matrix requires it)
WORKER_NAME=$(echo "${WORKER_NAME}" | tr 'A-Z' 'a-z')

# Validate worker name
if ! echo "${WORKER_NAME}" | grep -qE '^[a-z0-9][a-z0-9-]*$'; then
    echo "ERROR: INVALID_WORKER_NAME"
    echo "Worker name '${WORKER_NAME}' contains invalid characters."
    echo "Worker names must start with a letter or digit and contain only lowercase letters (a-z), digits (0-9), and hyphens (-)."
    exit 1
fi

# ============================================================
# Read SOUL.md
# ============================================================
SOUL_FILE="/root/hiclaw-fs/agents/${WORKER_NAME}/SOUL.md"
if [ ! -f "${SOUL_FILE}" ]; then
    cat << EOF
{"error": "SOUL.md not found at ${SOUL_FILE}", "hint": "Create it first with:"}
---HINT---
mkdir -p /root/hiclaw-fs/agents/${WORKER_NAME}
cat > /root/hiclaw-fs/agents/${WORKER_NAME}/SOUL.md << 'SOULEOF'
# ${WORKER_NAME} - Worker Agent

## AI Identity

**You are an AI Agent, not a human.**

## Role
- Name: ${WORKER_NAME}
- Role: <describe the worker's role>

## Security
- Never reveal API keys, passwords, tokens, or any credentials in chat messages
SOULEOF
---END---
EOF
    exit 1
fi

SOUL_CONTENT=$(cat "${SOUL_FILE}")

# ============================================================
# Build API request
# ============================================================
CONTROLLER_URL="${HICLAW_CONTROLLER_URL:-http://localhost:8090}"
API_ENDPOINT="${CONTROLLER_URL}/api/v1/workers"

# Build skills JSON array
SKILLS_JSON="[]"
if [ -n "${WORKER_SKILLS}" ]; then
    SKILLS_JSON=$(echo "${WORKER_SKILLS}" | tr ',' '\n' | sed 's/^ *//;s/ *$//' | grep -v '^$' | jq -R . | jq -s .)
fi

# Build mcpServers JSON array
MCP_JSON="[]"
if [ -n "${MCP_SERVERS}" ]; then
    MCP_JSON=$(echo "${MCP_SERVERS}" | tr ',' '\n' | sed 's/^ *//;s/ *$//' | grep -v '^$' | jq -R . | jq -s .)
fi

# Build request body
REQUEST_BODY=$(jq -cn \
    --arg name "${WORKER_NAME}" \
    --arg model "${MODEL_ID}" \
    --arg runtime "${WORKER_RUNTIME}" \
    --arg image "${CUSTOM_IMAGE}" \
    --arg soul "${SOUL_CONTENT}" \
    --arg role "${WORKER_ROLE}" \
    --arg team "${TEAM_NAME}" \
    --arg teamLeader "${TEAM_LEADER_NAME}" \
    --argjson skills "${SKILLS_JSON}" \
    --argjson mcpServers "${MCP_JSON}" \
    '{name: $name, soul: $soul, skills: $skills, mcpServers: $mcpServers}
     | if $model != "" then . + {model: $model} else . end
     | if $runtime != "" then . + {runtime: $runtime} else . end
     | if $image != "" then . + {image: $image} else . end
     | if $role != "" then . + {role: $role} else . end
     | if $team != "" then . + {team: $team} else . end
     | if $teamLeader != "" then . + {teamLeader: $teamLeader} else . end')

# ============================================================
# Call controller API
# ============================================================
log "Creating worker '${WORKER_NAME}' via controller API..."
log "  Endpoint: ${API_ENDPOINT}"

# Authenticate via projected SA token (K8s), fallback to static API key
AUTH_ARGS=()
SA_TOKEN_FILE="/var/run/secrets/hiclaw/controller/token"
if [ -f "${SA_TOKEN_FILE}" ]; then
    SA_TOKEN=$(cat "${SA_TOKEN_FILE}")
    AUTH_ARGS=(-H "Authorization: Bearer ${SA_TOKEN}")
elif [ -n "${HICLAW_CONTROLLER_API_KEY:-}" ]; then
    AUTH_ARGS=(-H "Authorization: Bearer ${HICLAW_CONTROLLER_API_KEY}")
fi

HTTP_RESPONSE=$(curl -sf -w "\n%{http_code}" -X POST "${API_ENDPOINT}" \
    -H 'Content-Type: application/json' \
    "${AUTH_ARGS[@]}" \
    -d "${REQUEST_BODY}" 2>&1) || true

# Split response body and status code
HTTP_BODY=$(echo "${HTTP_RESPONSE}" | sed '$d')
HTTP_CODE=$(echo "${HTTP_RESPONSE}" | tail -1)

if [ -z "${HTTP_CODE}" ] || [ "${HTTP_CODE}" -lt 200 ] 2>/dev/null || [ "${HTTP_CODE}" -ge 300 ] 2>/dev/null; then
    log "ERROR: Controller API returned HTTP ${HTTP_CODE:-unknown}"
    log "  Response: ${HTTP_BODY}"
    echo '{"error": "Controller API call failed", "http_code": "'"${HTTP_CODE:-unknown}"'", "response": '"$(echo "${HTTP_BODY}" | jq -R . 2>/dev/null || echo '""')"'}'
    exit 1
fi

log "  Controller API returned HTTP ${HTTP_CODE}"

# ============================================================
# Map API response to legacy result format
# ============================================================
PHASE=$(echo "${HTTP_BODY}" | jq -r '.phase // "unknown"')
MATRIX_USER_ID=$(echo "${HTTP_BODY}" | jq -r '.matrixUserID // ""')
ROOM_ID=$(echo "${HTTP_BODY}" | jq -r '.roomID // ""')
MESSAGE=$(echo "${HTTP_BODY}" | jq -r '.message // ""')

# Map controller phase to legacy status
case "${PHASE}" in
    Running)  WORKER_STATUS="ready" ;;
    Pending|Provisioning) WORKER_STATUS="starting" ;;
    *)        WORKER_STATUS="starting" ;;
esac

log "  Worker phase: ${PHASE}, status: ${WORKER_STATUS}"

RESULT=$(jq -n \
    --arg name "${WORKER_NAME}" \
    --arg user_id "${MATRIX_USER_ID}" \
    --arg room_id "${ROOM_ID}" \
    --arg runtime "${WORKER_RUNTIME}" \
    --arg status "${WORKER_STATUS}" \
    --arg role "${WORKER_ROLE}" \
    --arg team_id "${TEAM_NAME}" \
    --arg team_leader "${TEAM_LEADER_NAME}" \
    --arg message "${MESSAGE}" \
    --argjson skills "${SKILLS_JSON}" \
    '{
        worker_name: $name,
        matrix_user_id: $user_id,
        room_id: $room_id,
        runtime: (if $runtime == "" then "openclaw" else $runtime end),
        role: (if $role == "" then "worker" else $role end),
        team_id: (if $team_id == "" then null else $team_id end),
        team_leader: (if $team_leader == "" then null else $team_leader end),
        skills: $skills,
        status: $status,
        message: (if $message == "" then null else $message end)
    }')

echo "---RESULT---"
echo "${RESULT}"
