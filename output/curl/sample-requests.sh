#!/usr/bin/env bash
# =============================================================================
# Movie Discovery Platform — Sample cURL Requests
# =============================================================================
# All requests go through the API Gateway on port 8080.
# A mock Bearer token is used for authentication.
# =============================================================================

BASE_URL="http://localhost:8080"
TOKEN="Bearer test-token-12345"

echo "============================================="
echo " Movie Discovery Platform - Sample Requests"
echo "============================================="
echo ""

# ─────────────────────────────────────────────
# 1. Health Checks (no auth required)
# ─────────────────────────────────────────────
echo ">>> 1. Gateway Health Check"
curl -s "${BASE_URL}/health" | python3 -m json.tool
echo ""

echo ">>> 2. Movie Service Health Check (direct)"
curl -s "http://localhost:8081/api/v1/health" | python3 -m json.tool
echo ""

echo ">>> 3. User Preference Service Health Check (direct)"
curl -s "http://localhost:8082/api/v1/health" | python3 -m json.tool
echo ""

echo ">>> 4. Recommendation Service Health Check (direct)"
curl -s "http://localhost:8083/api/v1/health" | python3 -m json.tool
echo ""

# ─────────────────────────────────────────────
# 2. Sync Movies from TMDB
# ─────────────────────────────────────────────
echo ">>> 5. Sync movies from TMDB (5 pages)"
curl -s -X POST "${BASE_URL}/api/v1/admin/sync?pages=5" \
  -H "Authorization: ${TOKEN}" | python3 -m json.tool
echo ""

# ─────────────────────────────────────────────
# 3. Movie Endpoints
# ─────────────────────────────────────────────
echo ">>> 6. List movies (page 1, sorted by popularity)"
curl -s "${BASE_URL}/api/v1/movies?page=1&page_size=5&sort_by=popularity&order=desc" \
  -H "Authorization: ${TOKEN}" | python3 -m json.tool
echo ""

echo ">>> 7. List movies with date filter"
curl -s "${BASE_URL}/api/v1/movies?release_date_from=2024-01-01&release_date_to=2025-12-31&page_size=5" \
  -H "Authorization: ${TOKEN}" | python3 -m json.tool
echo ""

echo ">>> 8. Get movie detail (ID=1)"
curl -s "${BASE_URL}/api/v1/movies/1" \
  -H "Authorization: ${TOKEN}" | python3 -m json.tool
echo ""

# ─────────────────────────────────────────────
# 4. User Management
# ─────────────────────────────────────────────
echo ">>> 9. Create a new user"
curl -s -X POST "${BASE_URL}/api/v1/users" \
  -H "Authorization: ${TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{"username": "johndoe", "email": "john@example.com"}' | python3 -m json.tool
echo ""

echo ">>> 10. Get user info (ID=1)"
curl -s "${BASE_URL}/api/v1/users/1" \
  -H "Authorization: ${TOKEN}" | python3 -m json.tool
echo ""

# ─────────────────────────────────────────────
# 5. User Preferences
# ─────────────────────────────────────────────
echo ">>> 11. Set user preferences"
curl -s -X POST "${BASE_URL}/api/v1/users/1/preferences" \
  -H "Authorization: ${TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "preferred_genres": ["Action", "Science Fiction", "Adventure"],
    "preferred_language": "en",
    "min_rating": 7.0
  }' | python3 -m json.tool
echo ""

echo ">>> 12. Get user preferences"
curl -s "${BASE_URL}/api/v1/users/1/preferences" \
  -H "Authorization: ${TOKEN}" | python3 -m json.tool
echo ""

# ─────────────────────────────────────────────
# 6. User Interactions
# ─────────────────────────────────────────────
echo ">>> 13. Record a 'like' interaction"
curl -s -X POST "${BASE_URL}/api/v1/users/1/interactions" \
  -H "Authorization: ${TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{"movie_id": 1, "interaction_type": "like"}' | python3 -m json.tool
echo ""

echo ">>> 14. Record a 'watchlist' interaction"
curl -s -X POST "${BASE_URL}/api/v1/users/1/interactions" \
  -H "Authorization: ${TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{"movie_id": 2, "interaction_type": "watchlist"}' | python3 -m json.tool
echo ""

echo ">>> 15. Get user interactions"
curl -s "${BASE_URL}/api/v1/users/1/interactions" \
  -H "Authorization: ${TOKEN}" | python3 -m json.tool
echo ""

# ─────────────────────────────────────────────
# 7. Recommendations
# ─────────────────────────────────────────────
echo ">>> 16. Get recommendations for user 1 (top 10)"
curl -s "${BASE_URL}/api/v1/users/1/recommendations?limit=10" \
  -H "Authorization: ${TOKEN}" | python3 -m json.tool
echo ""

echo ">>> 17. Get recommendation rules"
curl -s "${BASE_URL}/api/v1/rules" \
  -H "Authorization: ${TOKEN}" | python3 -m json.tool
echo ""

echo "============================================="
echo " All requests completed."
echo "============================================="
