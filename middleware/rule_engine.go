package middleware

import (
	"log"

	"github.com/umesh/dgla/router"
)

// Condition represents a rule condition
type Condition struct {
	If string `yaml:"if"`
}

// Action represents an action to take when a rule condition is met
type Action struct {
	Ensure string `yaml:"ensure"`
}

// ViolationResponse represents actions to take when a rule is violated
type ViolationResponse struct {
	BlockTransfer bool   `yaml:"block_transfer"`
	Alert         string `yaml:"alert,omitempty"`
}

// Rule represents a data governance rule
type Rule struct {
	RuleID           string           `yaml:"rule_id"`
	Condition        Condition        `yaml:"condition"`
	Actions          []Action         `yaml:"actions"`
	ViolationResponse ViolationResponse `yaml:"violation_response"`
}

// RuleEngine evaluates rules against data requests
type RuleEngine struct {
	rules []Rule
}

// NewRuleEngine creates a new rule engine with the given rules
func NewRuleEngine(rules []Rule) *RuleEngine {
	return &RuleEngine{
		rules: rules,
	}
}

// Evaluate checks if a data request complies with the rules
func (e *RuleEngine) Evaluate(request router.DataRequest) (bool, []ViolationResponse) {
	violations := []ViolationResponse{}
	
	for _, rule := range e.rules {
		if e.evaluateCondition(rule.Condition, request) {
			// Check if all actions pass
			compliant := true
			for _, action := range rule.Actions {
				if !e.evaluateAction(action, request) {
					compliant = false
					break
				}
			}
			
			if !compliant {
				log.Printf("Rule %s violated by request %s", rule.RuleID, request.JobID)
				violations = append(violations, rule.ViolationResponse)
			}
		}
	}
	
	return len(violations) == 0, violations
}

// evaluateCondition checks if a condition applies to a request
func (e *RuleEngine) evaluateCondition(condition Condition, request router.DataRequest) bool {
	// This is a simplified version - in a real implementation, 
	// we would use a proper expression evaluator instead of hardcoding
	
	switch condition.If {
	case "data.region == 'EU' and data.is_pii == true":
		return request.Region == "EU" && request.IsPII
	case "data.region == 'US' and data.is_pii == true":
		return request.Region == "US" && request.IsPII
	default:
		log.Printf("Unknown condition: %s", condition.If)
		return false
	}
}

// evaluateAction checks if an action is compliant
func (e *RuleEngine) evaluateAction(action Action, request router.DataRequest) bool {
	// This is a simplified version - in a real implementation,
	// we would use a proper expression evaluator
	
	switch action.Ensure {
	case "destination.region == 'EU'":
		return request.Destination == "EU"
	case "destination.region == 'US'":
		return request.Destination == "US"
	case "anonymization_time <= 5 minutes":
		// For demonstration purposes, assume true
		return true
	default:
		log.Printf("Unknown action: %s", action.Ensure)
		return false
	}
}

// HandleViolations processes violation responses
func (e *RuleEngine) HandleViolations(violations []ViolationResponse) {
	for _, violation := range violations {
		if violation.BlockTransfer {
			log.Println("BLOCKING DATA TRANSFER due to rule violation")
		}
		
		if violation.Alert != "" {
			log.Printf("ALERT: Notifying %s of rule violation", violation.Alert)
			// In a real implementation, this would send an actual notification
		}
	}
}
