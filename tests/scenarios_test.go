package tests

import (
	"testing"
	"time"

	"github.com/umesh/dgla/cache"
	"github.com/umesh/dgla/merkle"
	"github.com/umesh/dgla/middleware"
	"github.com/umesh/dgla/router"
)

// Test rules
var testRules = []middleware.Rule{
	{
		RuleID: "EU_PII_REGION_LOCK",
		Condition: middleware.Condition{
			If: "data.region == 'EU' and data.is_pii == true",
		},
		Actions: []middleware.Action{
			{Ensure: "destination.region == 'EU'"},
		},
		ViolationResponse: middleware.ViolationResponse{
			BlockTransfer: true,
			Alert:         "DataPrivacyTeam",
		},
	},
}

// TestCompliantEUFlow tests a compliant EU data flow
func TestCompliantEUFlow(t *testing.T) {
	// Initialize components
	c := cache.NewRedisLikeCache()
	identityRouter := router.NewIdentityRouter(c)
	ruleEngine := middleware.NewRuleEngine(testRules)

	// Create a compliant EU request
	request := router.DataRequest{
		JobID:       "MLModel123_EU",
		DataAsset:   "customer_pii_table",
		Region:      "EU",
		Action:      "read",
		IsPII:       true,
		Timestamp:   time.Now(),
		Source:      "database",
		Destination: "EU", // Compliant: EU PII stays in EU
		Metadata: map[string]interface{}{
			"purpose": "model_training",
		},
	}

	// Evaluate rules
	compliant, violations := ruleEngine.Evaluate(request)
	if !compliant {
		t.Errorf("Expected request to be compliant, but got violations: %+v", violations)
	}

	// Route the request
	err := identityRouter.Route(request)
	if err != nil {
		t.Errorf("Error routing request: %v", err)
	}

	// Generate Merkle proof
	dataItems := []interface{}{request}
	tree, err := merkle.NewMerkleTree(dataItems)
	if err != nil {
		t.Errorf("Error creating Merkle tree: %v", err)
	}

	proof := tree.GenerateProof()
	if proof.Root == "" {
		t.Error("Expected Merkle root to be non-empty")
	}

	// Verify the flow was tracked
	logs := identityRouter.GetLogs()
	if len(logs) != 1 {
		t.Errorf("Expected 1 log entry, got %d", len(logs))
	}
}

// TestNonCompliantEUToUSFlow tests a non-compliant flow (EU PII to US)
func TestNonCompliantEUToUSFlow(t *testing.T) {
	// Initialize components
	c := cache.NewRedisLikeCache()
	identityRouter := router.NewIdentityRouter(c)
	ruleEngine := middleware.NewRuleEngine(testRules)

	// Create a non-compliant request (EU PII to US)
	request := router.DataRequest{
		JobID:       "MLModel123_EU_US",
		DataAsset:   "customer_pii_table",
		Region:      "EU",
		Action:      "read",
		IsPII:       true,
		Timestamp:   time.Now(),
		Source:      "database",
		Destination: "US", // Violation: EU PII going to US
		Metadata: map[string]interface{}{
			"purpose": "model_training",
		},
	}

	// Evaluate rules
	compliant, violations := ruleEngine.Evaluate(request)
	if compliant {
		t.Error("Expected request to be non-compliant, but no violations were found")
	}

	// Check if violations have the right response
	if len(violations) != 1 {
		t.Errorf("Expected 1 violation, got %d", len(violations))
	}

	if !violations[0].BlockTransfer {
		t.Error("Expected transfer to be blocked, but it wasn't")
	}
	
	// Check that the router logs the request even though there's a violation
	// In a real system, we might want to prevent logging for blocked requests
	// but for this test, we'll log it to verify the router is working
	identityRouter.Route(request)
	logs := identityRouter.GetLogs()
	if len(logs) != 1 {
		t.Errorf("Expected 1 log entry, got %d", len(logs))
	}
}
