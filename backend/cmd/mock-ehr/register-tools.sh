#!/bin/bash
# Register mock EHR tools with the AI agent registry
# Usage: ./register-tools.sh [API_BASE_URL]

API_BASE="${1:-http://localhost:8080}"
REGISTRY_URL="${API_BASE}/api/v1/ai/registry/tools"
EHR_BASE="http://localhost:8085"

echo "Registering EHR tools with ${REGISTRY_URL}..."
echo "EHR API at ${EHR_BASE}"
echo ""

# 1. create_patient - requires confirmation
echo "1. Registering create_patient..."
curl -s -X POST "$REGISTRY_URL" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: default-tenant" \
  -d '{
    "app_name": "ehr",
    "tool_name": "create_patient",
    "description": "Register a new patient in the EHR system. Use this when the user wants to create or register a new patient. Ask for at least the patient name before calling this tool.",
    "parameters": {
      "type": "object",
      "properties": {
        "name":    { "type": "string", "description": "Patient full name (required)" },
        "dob":     { "type": "string", "description": "Date of birth in YYYY-MM-DD format" },
        "gender":  { "type": "string", "enum": ["male", "female", "other"], "description": "Patient gender" },
        "phone":   { "type": "string", "description": "Phone number" },
        "email":   { "type": "string", "description": "Email address" },
        "address": { "type": "string", "description": "Home address" }
      },
      "required": ["name"]
    },
    "execution": {
      "type": "http",
      "method": "POST",
      "url": "'"${EHR_BASE}"'/patients",
      "timeout_ms": 10000
    },
    "requires_confirmation": true
  }' | python -m json.tool 2>/dev/null || echo "(curl output above)"
echo ""

# 2. get_patient - no confirmation
echo "2. Registering get_patient..."
curl -s -X POST "$REGISTRY_URL" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: default-tenant" \
  -d '{
    "app_name": "ehr",
    "tool_name": "get_patient",
    "description": "Get detailed information about a specific patient by their patient ID. Use this when you need to look up a specific patient record.",
    "parameters": {
      "type": "object",
      "properties": {
        "id": { "type": "string", "description": "Patient ID (e.g., P-1001)" }
      },
      "required": ["id"]
    },
    "execution": {
      "type": "http",
      "method": "GET",
      "url": "'"${EHR_BASE}"'/patients/{id}",
      "timeout_ms": 5000
    },
    "requires_confirmation": false
  }' | python -m json.tool 2>/dev/null || echo "(curl output above)"
echo ""

# 3. search_patients - no confirmation
echo "3. Registering search_patients..."
curl -s -X POST "$REGISTRY_URL" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: default-tenant" \
  -d '{
    "app_name": "ehr",
    "tool_name": "search_patients",
    "description": "Search for patients by name or ID. Use this when the user wants to find a patient but does not know the exact patient ID.",
    "parameters": {
      "type": "object",
      "properties": {
        "q": { "type": "string", "description": "Search query - patient name or ID" }
      },
      "required": ["q"]
    },
    "execution": {
      "type": "http",
      "method": "GET",
      "url": "'"${EHR_BASE}"'/patients/search",
      "timeout_ms": 5000
    },
    "requires_confirmation": false
  }' | python -m json.tool 2>/dev/null || echo "(curl output above)"
echo ""

# 4. get_patient_history - no confirmation
echo "4. Registering get_patient_history..."
curl -s -X POST "$REGISTRY_URL" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: default-tenant" \
  -d '{
    "app_name": "ehr",
    "tool_name": "get_patient_history",
    "description": "Get the medical history for a patient including visits, lab results, prescriptions, and diagnoses. Use this when you need to review or summarize a patient'\''s medical history.",
    "parameters": {
      "type": "object",
      "properties": {
        "patient_id": { "type": "string", "description": "Patient ID (e.g., P-1001)" }
      },
      "required": ["patient_id"]
    },
    "execution": {
      "type": "http",
      "method": "GET",
      "url": "'"${EHR_BASE}"'/history/{patient_id}",
      "timeout_ms": 5000
    },
    "requires_confirmation": false
  }' | python -m json.tool 2>/dev/null || echo "(curl output above)"
echo ""

# 5. list_patients - no confirmation
echo "5. Registering list_patients..."
curl -s -X POST "$REGISTRY_URL" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: default-tenant" \
  -d '{
    "app_name": "ehr",
    "tool_name": "list_patients",
    "description": "List all patients in the EHR system. Use this when the user wants to see all registered patients.",
    "parameters": {
      "type": "object",
      "properties": {}
    },
    "execution": {
      "type": "http",
      "method": "GET",
      "url": "'"${EHR_BASE}"'/patients",
      "timeout_ms": 5000
    },
    "requires_confirmation": false
  }' | python -m json.tool 2>/dev/null || echo "(curl output above)"
echo ""

echo "Done! 5 EHR tools registered."
echo ""
echo "Test queries to try in the chat:"
echo "  - \"Show me all patients\""
echo "  - \"Find patient John Smith\""
echo "  - \"Register a new patient named Jane Doe, born 1995-06-20\""
echo "  - \"Show me the medical history for patient P-1001\""
echo "  - \"Summarize John Smith's health status\""
