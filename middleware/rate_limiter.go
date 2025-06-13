package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/patrickmn/go-cache"
	"golang.org/x/time/rate"
)

// RateLimitTier defines API usage tiers
type RateLimitTier string

const (
	TierStarter   RateLimitTier = "starter"
	TierPro       RateLimitTier = "pro"
	TierEnterprise RateLimitTier = "enterprise"
)

// tierLimits defines requests-per-minute for each tier
var tierLimits = map[RateLimitTier]int{
	TierStarter:   60,   // 1 req/sec
	TierPro:       300,  // 5 req/sec
	TierEnterprise: 1200, // 20 req/sec
}

// RateLimiter implements rate limiting by IP address and API key
type RateLimiter struct {
	ipLimiters     *cache.Cache
	apiKeyLimiters *cache.Cache
	mu             sync.Mutex
	metrics        MetricsRecorder
}

// MetricsRecorder records rate limiting metrics
type MetricsRecorder interface {
	RecordRateLimitCheck(endpoint string)
	RecordRateLimitExceeded(endpoint, clientIP string)
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(metrics MetricsRecorder) *RateLimiter {
	// Cache with 5 minute expiration, cleanup every minute
	ipCache := cache.New(5*time.Minute, time.Minute)
	apiKeyCache := cache.New(5*time.Minute, time.Minute)
	
	return &RateLimiter{
		ipLimiters:     ipCache,
		apiKeyLimiters: apiKeyCache,
		metrics:        metrics,
	}
}

// getIPLimiter returns a rate limiter for an IP address
func (rl *RateLimiter) getIPLimiter(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Default IP rate limit is 60 requests per minute (1 req/sec)
	limiter, found := rl.ipLimiters.Get(ip)
	if !found {
		// Create a new limiter with 1 req/sec and burst of 5
		limiter = rate.NewLimiter(rate.Limit(1), 5)
		rl.ipLimiters.Set(ip, limiter, cache.DefaultExpiration)
	}
	
	return limiter.(*rate.Limiter)
}

// getAPIKeyLimiter returns a rate limiter for an API key
func (rl *RateLimiter) getAPIKeyLimiter(apiKey string, tier RateLimitTier) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	limiter, found := rl.apiKeyLimiters.Get(apiKey)
	if !found {
		// Get limit for tier or default to starter
		limit, ok := tierLimits[tier]
		if !ok {
			limit = tierLimits[TierStarter]
		}
		
		// Convert to requests per second
		rps := float64(limit) / 60.0
		burst := limit / 10
		if burst < 5 {
			burst = 5
		}
		
		// Create a new limiter with the tier's rate limit and appropriate burst
		limiter = rate.NewLimiter(rate.Limit(rps), burst)
		rl.apiKeyLimiters.Set(apiKey, limiter, cache.DefaultExpiration)
	}
	
	return limiter.(*rate.Limiter)
}

// getTierFromRequest determines the user's tier from token claims or headers
func (rl *RateLimiter) getTierFromRequest(r *http.Request) RateLimitTier {
	// Try to get from context (added by auth middleware)
	if tier, ok := r.Context().Value("tier").(string); ok {
		return RateLimitTier(tier)
	}
	
	// Try to get from header
	if tier := r.Header.Get("X-API-Tier"); tier != "" {
		return RateLimitTier(tier)
	}
	
	// Default to starter tier
	return TierStarter
}

// getAPIKeyFromRequest extracts API key from request
func (rl *RateLimiter) getAPIKeyFromRequest(r *http.Request) string {
	// Try header first
	apiKey := r.Header.Get("X-API-Key")
	if apiKey != "" {
		return apiKey
	}
	
	// Then query param
	apiKey = r.URL.Query().Get("api_key")
	if apiKey != "" {
		return apiKey
	}
	
	// If no API key, fall back to IP-based limiting
	return ""
}

// RateLimit is middleware that limits request frequency
func (rl *RateLimiter) RateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Record metric for rate limit check
		if rl.metrics != nil {
			rl.metrics.RecordRateLimitCheck(r.URL.Path)
		}
		
		clientIP := r.RemoteAddr
		apiKey := rl.getAPIKeyFromRequest(r)
		
		var allowed bool
		
		if apiKey != "" {
			// Use API key-based rate limiting with tier
			tier := rl.getTierFromRequest(r)
			limiter := rl.getAPIKeyLimiter(apiKey, tier)
			allowed = limiter.Allow()
		} else {
			// Fall back to IP-based rate limiting
			limiter := rl.getIPLimiter(clientIP)
			allowed = limiter.Allow()
		}
		
		if !allowed {
			// Record metric for rate limit exceeded
			if rl.metrics != nil {
				rl.metrics.RecordRateLimitExceeded(r.URL.Path, clientIP)
			}
			
			w.Header().Add("Retry-After", "60")
			http.Error(w, "Rate limit exceeded. Please try again later.", http.StatusTooManyRequests)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}
