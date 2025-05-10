package agreements

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/umesh/dgla/middleware"
	"github.com/umesh/dgla/merkle"
	"gopkg.in/yaml.v2"
)

// Agreement represents a data governance agreement with rules and proof
type Agreement struct {
	AgreementID string             `yaml:"agreement_id"`
	Rules       []middleware.Rule  `yaml:"rules"`
	Proof       merkle.MerkleProof `yaml:"proof,omitempty"`
}

// ParseAgreement loads and parses an agreement from a YAML file
func ParseAgreement(filePath string) (*Agreement, error) {
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	var agreement Agreement
	err = yaml.Unmarshal(data, &agreement)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling YAML: %w", err)
	}

	return &agreement, nil
}

// SaveAgreement writes an agreement to a YAML file
func SaveAgreement(agreement *Agreement, filePath string) error {
	data, err := yaml.Marshal(agreement)
	if err != nil {
		return fmt.Errorf("error marshalling YAML: %w", err)
	}

	err = ioutil.WriteFile(filePath, data, 0644)
	if err != nil {
		return fmt.Errorf("error writing file: %w", err)
	}

	return nil
}

// IsAgreementFile checks if a file is a merkle#secret.yaml format
func IsAgreementFile(filePath string) bool {
	return strings.HasSuffix(filePath, "merkle#secret.yaml")
}

// UpdateProof updates the Merkle proof in an agreement
func UpdateProof(agreement *Agreement, proof merkle.MerkleProof) {
	agreement.Proof = proof
}

// CreateAgreementFile creates a new agreement file with rules
func CreateAgreementFile(agreement *Agreement, filePath string) error {
	// Check if file exists
	if _, err := os.Stat(filePath); err == nil {
		return fmt.Errorf("file already exists: %s", filePath)
	}

	// Create the file
	return SaveAgreement(agreement, filePath)
}

// GetRules extracts rules from an agreement
func GetRules(agreement *Agreement) []middleware.Rule {
	return agreement.Rules
}
