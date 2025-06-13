package auth

import (
	"errors"
	"net/http"
)

// Role types
const (
	RoleVerifier = "verifier" // can POST /verify
	RoleAuditor  = "auditor"  // can GET /logs
	RoleAdmin    = "admin"    // can POST /rules, /anchor
)

// RoleAccessMap defines endpoint access permissions for each role
var RoleAccessMap = map[string]map[string][]string{
	RoleVerifier: {
		http.MethodPost: {"/verify"},
		http.MethodGet:  {"/health", "/ready", "/alive", "/metrics", "/docs"},
	},
	RoleAuditor: {
		http.MethodGet: {"/logs", "/export", "/health", "/ready", "/alive", "/metrics", "/docs"},
	},
	RoleAdmin: {
		http.MethodPost: {"/verify", "/rules", "/anchor", "/auth/login"},
		http.MethodGet:  {"/logs", "/export", "/health", "/ready", "/alive", "/metrics", "/docs"},
	},
}

// RoleAuthMiddleware provides role-based access control
func (m *JWTManager) RoleAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip auth if not enabled
		if !m.config.Enabled {
			next.ServeHTTP(w, r)
			return
		}

		// Extract token
		tokenString, err := ExtractTokenFromRequest(r)
		if err != nil {
			http.Error(w, "Unauthorized: "+err.Error(), http.StatusUnauthorized)
			return
		}

		// Validate token
		claims, err := m.ValidateToken(tokenString)
		if err != nil {
			http.Error(w, "Unauthorized: "+err.Error(), http.StatusUnauthorized)
			return
		}

		// Get user role
		role := claims.Role
		if role == "" {
			http.Error(w, "Unauthorized: missing role claim", http.StatusUnauthorized)
			return
		}

		// Check if role exists in the access map
		roleAccess, exists := RoleAccessMap[role]
		if !exists {
			http.Error(w, "Unauthorized: invalid role", http.StatusUnauthorized)
			return
		}

		// Check if the method is allowed for this role
		methodAccess, exists := roleAccess[r.Method]
		if !exists {
			http.Error(w, "Unauthorized: method not allowed for this role", http.StatusForbidden)
			return
		}

		// Check if the path is allowed for this role and method
		path := r.URL.Path
		allowed := false
		for _, allowedPath := range methodAccess {
			if path == allowedPath {
				allowed = true
				break
			}
		}

		if !allowed {
			http.Error(w, "Unauthorized: endpoint not allowed for this role", http.StatusForbidden)
			return
		}

		// Add claims to request context
		ctx := r.Context()
		ctx = ContextWithUserClaims(ctx, claims)

		// Call the next handler with the updated context
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// ContextWithUserClaims adds user claims to the context
func ContextWithUserClaims(ctx http.Context, claims *UserClaims) http.Context {
	ctx = context.WithValue(ctx, "user_id", claims.UserID)
	ctx = context.WithValue(ctx, "username", claims.Username)
	ctx = context.WithValue(ctx, "role", claims.Role)
	return ctx
}

// HasRole checks if the request has the required role
func HasRole(r *http.Request, role string) bool {
	userRole, ok := r.Context().Value("role").(string)
	if !ok {
		return false
	}
	return userRole == role || userRole == RoleAdmin
}

// LoginHandler creates a handler for user login
func (m *JWTManager) LoginHandler() http.HandlerFunc {
	type LoginRequest struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Role     string `json:"role"`
	}

	type LoginResponse struct {
		Token string `json:"token"`
		Role  string `json:"role"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		// Only accept POST requests
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Parse request body
		var req LoginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request format", http.StatusBadRequest)
			return
		}

		// Validate user credentials
		// In a real application, this would check against a database
		if !validateUserCredentials(req.Username, req.Password) {
			http.Error(w, "Invalid credentials", http.StatusUnauthorized)
			return
		}

		// Validate requested role
		if !isValidRole(req.Role) {
			http.Error(w, "Invalid role", http.StatusBadRequest)
			return
		}

		// Generate a token
		userID := generateUserID(req.Username)
		token, err := m.GenerateToken(userID, req.Username, req.Role)
		if err != nil {
			http.Error(w, "Failed to generate token", http.StatusInternalServerError)
			return
		}

		// Return the token
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(LoginResponse{
			Token: token,
			Role:  req.Role,
		})
	}
}

// Helper function to validate user credentials (mock implementation)
func validateUserCredentials(username, password string) bool {
	// In a real application, this would check against a database
	// This is just a placeholder
	return username != "" && password != ""
}

// Helper function to validate role
func isValidRole(role string) bool {
	switch role {
	case RoleVerifier, RoleAuditor, RoleAdmin:
		return true
	default:
		return false
	}
}

// Helper function to generate a user ID
func generateUserID(username string) string {
	// In a real application, this would be more sophisticated
	return "user_" + username
}
