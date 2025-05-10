package benchmark

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/umesh/dgla/router"
)

// WorkloadPattern defines different patterns of data governance requests
type WorkloadPattern string

const (
	// StandardWorkload represents a typical mix of governance requests
	StandardWorkload WorkloadPattern = "standard"
	
	// ComplianceAuditWorkload simulates an intensive compliance audit
	ComplianceAuditWorkload WorkloadPattern = "compliance_audit"
	
	// DataMigrationWorkload simulates a large data migration scenario
	DataMigrationWorkload WorkloadPattern = "data_migration"
	
	// RegulatoryInspectionWorkload simulates a regulatory inspection scenario
	RegulatoryInspectionWorkload WorkloadPattern = "regulatory_inspection"
)

// WorkloadGenerator creates realistic data governance workloads
type WorkloadGenerator struct {
	pattern     WorkloadPattern
	regions     []string
	dataAssets  []string
	sources     []string
	destinations []string
	actions     []string
}

// NewWorkloadGenerator creates a new workload generator with specified pattern
func NewWorkloadGenerator(pattern WorkloadPattern) *WorkloadGenerator {
	rand.Seed(time.Now().UnixNano())
	
	return &WorkloadGenerator{
		pattern: pattern,
		regions: []string{"EU", "US", "APAC", "UK", "LATAM"},
		dataAssets: []string{
			"customer_pii", 
			"payment_details", 
			"healthcare_records", 
			"financial_transactions", 
			"employee_data",
			"marketing_analytics",
			"product_metadata",
			"operational_metrics",
		},
		sources: []string{
			"mysql_db", 
			"postgres_db", 
			"mongodb_collection", 
			"s3_bucket", 
			"azure_blob",
			"kafka_topic",
			"hadoop_cluster",
			"snowflake_warehouse",
		},
		destinations: []string{
			"analytics_dashboard", 
			"reporting_db", 
			"data_lake", 
			"ml_training", 
			"archive",
			"regulatory_exports",
			"customer_portal",
			"third_party_integration",
		},
		actions: []string{
			"read",
			"write",
			"transform",
			"aggregate",
			"anonymize",
			"encrypt",
			"export",
			"delete",
		},
	}
}

// GenerateWorkload creates a set of data requests based on the selected pattern
func (wg *WorkloadGenerator) GenerateWorkload(count int) []router.DataRequest {
	switch wg.pattern {
	case ComplianceAuditWorkload:
		return wg.generateComplianceAuditWorkload(count)
	case DataMigrationWorkload:
		return wg.generateDataMigrationWorkload(count)
	case RegulatoryInspectionWorkload:
		return wg.generateRegulatoryInspectionWorkload(count)
	default:
		return wg.generateStandardWorkload(count)
	}
}

// generateStandardWorkload creates a typical mix of governance requests
func (wg *WorkloadGenerator) generateStandardWorkload(count int) []router.DataRequest {
	requests := make([]router.DataRequest, count)
	
	// Standard workload: 70% compliant, 30% potentially non-compliant
	for i := 0; i < count; i++ {
		isPII := rand.Float32() < 0.6 // 60% of requests involve PII
		
		sourceRegion := wg.randomRegion()
		var destRegion string
		
		// For PII data, 30% chance of cross-region transfer (potentially non-compliant)
		if isPII && rand.Float32() < 0.3 {
			// Cross-region transfer
			destRegion = wg.randomRegionExcept(sourceRegion)
		} else {
			// Same region transfer (compliant)
			destRegion = sourceRegion
		}
		
		requests[i] = router.DataRequest{
			JobID:       fmt.Sprintf("job-%d-%s", i, time.Now().Format("150405")),
			DataAsset:   wg.randomDataAsset(),
			Region:      sourceRegion,
			Action:      wg.randomAction(),
			IsPII:       isPII,
			Timestamp:   time.Now(),
			Source:      wg.randomSource(),
			Destination: destRegion,
			Metadata: map[string]interface{}{
				"purpose":      wg.randomPurpose(),
				"user_id":      fmt.Sprintf("user-%d", rand.Intn(100)),
				"request_time": time.Now().Unix(),
			},
		}
	}
	
	return requests
}

// generateComplianceAuditWorkload creates requests that simulate a compliance audit
func (wg *WorkloadGenerator) generateComplianceAuditWorkload(count int) []router.DataRequest {
	requests := make([]router.DataRequest, count)
	
	// Compliance audit: Heavy emphasis on PII data (90%)
	for i := 0; i < count; i++ {
		isPII := rand.Float32() < 0.9 // 90% of requests involve PII
		
		sourceRegion := wg.randomRegion()
		
		// In audit scenario, we test more edge cases: 40% cross-region transfers
		var destRegion string
		if isPII && rand.Float32() < 0.4 {
			// Cross-region transfer
			destRegion = wg.randomRegionExcept(sourceRegion)
		} else {
			// Same region transfer
			destRegion = sourceRegion
		}
		
		requests[i] = router.DataRequest{
			JobID:       fmt.Sprintf("audit-%d-%s", i, time.Now().Format("150405")),
			DataAsset:   wg.randomSensitiveDataAsset(), // Emphasize sensitive data
			Region:      sourceRegion,
			Action:      wg.randomAuditAction(), // Special audit actions
			IsPII:       isPII,
			Timestamp:   time.Now(),
			Source:      wg.randomSource(),
			Destination: destRegion,
			Metadata: map[string]interface{}{
				"purpose":       "compliance_audit",
				"auditor_id":    fmt.Sprintf("auditor-%d", rand.Intn(10)),
				"audit_case_id": fmt.Sprintf("case-%d", rand.Intn(1000)),
				"regulation":    wg.randomRegulation(),
			},
		}
	}
	
	return requests
}

// generateDataMigrationWorkload creates requests simulating a large data migration
func (wg *WorkloadGenerator) generateDataMigrationWorkload(count int) []router.DataRequest {
	requests := make([]router.DataRequest, count)
	
	// Select specific source and destination systems for the migration
	sourceSystem := wg.randomSource()
	destSystem := wg.randomDestination()
	
	// Select migration source and destination regions
	sourceRegion := wg.randomRegion()
	destRegion := sourceRegion // Usually migrations happen within same region for compliance
	
	// Data migration: Consists of many similar requests for different data assets
	for i := 0; i < count; i++ {
		isPII := rand.Float32() < 0.5 // 50% PII data
		
		requests[i] = router.DataRequest{
			JobID:       fmt.Sprintf("migration-%d-%s", i, time.Now().Format("150405")),
			DataAsset:   wg.randomDataAsset(),
			Region:      sourceRegion,
			Action:      "migrate", // Most migration actions are the same
			IsPII:       isPII,
			Timestamp:   time.Now(),
			Source:      sourceSystem,
			Destination: destSystem,
			Metadata: map[string]interface{}{
				"purpose":        "system_migration",
				"migration_id":   "MIG-2025-001",
				"batch_number":   i / 100, // Group in batches
				"total_batches":  count / 100,
				"source_region":  sourceRegion,
				"target_region":  destRegion,
				"transform_type": wg.randomTransformationType(),
			},
		}
	}
	
	return requests
}

// generateRegulatoryInspectionWorkload creates requests for regulatory inspection
func (wg *WorkloadGenerator) generateRegulatoryInspectionWorkload(count int) []router.DataRequest {
	requests := make([]router.DataRequest, count)
	
	// Regulatory inspection focuses heavily on PII and cross-border transfers
	for i := 0; i < count; i++ {
		// Almost all are PII in regulatory scenarios
		isPII := rand.Float32() < 0.95
		
		// Focus on specific regulated regions
		sourceRegion := wg.randomRegulatedRegion()
		
		// 60% cross-border transfers to test compliance
		var destRegion string
		if rand.Float32() < 0.6 {
			destRegion = wg.randomRegionExcept(sourceRegion)
		} else {
			destRegion = sourceRegion
		}
		
		requests[i] = router.DataRequest{
			JobID:       fmt.Sprintf("regulatory-%d-%s", i, time.Now().Format("150405")),
			DataAsset:   wg.randomRegulatoryDataAsset(),
			Region:      sourceRegion,
			Action:      wg.randomRegulatoryAction(),
			IsPII:       isPII,
			Timestamp:   time.Now(),
			Source:      wg.randomSource(),
			Destination: destRegion,
			Metadata: map[string]interface{}{
				"purpose":          "regulatory_compliance",
				"regulation":       wg.randomRegulation(),
				"authority":        wg.randomRegulatoryAuthority(),
				"inspection_id":    fmt.Sprintf("INSP-%d", rand.Intn(1000)),
				"data_category":    wg.randomDataCategory(),
				"retention_period": fmt.Sprintf("%d days", 30+rand.Intn(365)),
			},
		}
	}
	
	return requests
}

// Helper methods for generating random data elements

func (wg *WorkloadGenerator) randomRegion() string {
	return wg.regions[rand.Intn(len(wg.regions))]
}

func (wg *WorkloadGenerator) randomRegionExcept(excluded string) string {
	for {
		region := wg.regions[rand.Intn(len(wg.regions))]
		if region != excluded {
			return region
		}
	}
}

func (wg *WorkloadGenerator) randomRegulatedRegion() string {
	// Regions with strict data regulations
	regulatedRegions := []string{"EU", "UK", "APAC"}
	return regulatedRegions[rand.Intn(len(regulatedRegions))]
}

func (wg *WorkloadGenerator) randomDataAsset() string {
	return wg.dataAssets[rand.Intn(len(wg.dataAssets))]
}

func (wg *WorkloadGenerator) randomSensitiveDataAsset() string {
	sensitiveAssets := []string{
		"customer_pii", 
		"payment_details", 
		"healthcare_records", 
		"financial_transactions", 
		"employee_data",
	}
	return sensitiveAssets[rand.Intn(len(sensitiveAssets))]
}

func (wg *WorkloadGenerator) randomRegulatoryDataAsset() string {
	regulatoryAssets := []string{
		"customer_pii",
		"healthcare_records",
		"financial_transactions",
		"consent_records",
		"gdpr_subject_requests",
		"data_erasure_logs",
	}
	return regulatoryAssets[rand.Intn(len(regulatoryAssets))]
}

func (wg *WorkloadGenerator) randomSource() string {
	return wg.sources[rand.Intn(len(wg.sources))]
}

func (wg *WorkloadGenerator) randomDestination() string {
	return wg.destinations[rand.Intn(len(wg.destinations))]
}

func (wg *WorkloadGenerator) randomAction() string {
	return wg.actions[rand.Intn(len(wg.actions))]
}

func (wg *WorkloadGenerator) randomAuditAction() string {
	auditActions := []string{
		"read",
		"export",
		"verify",
		"audit",
		"inspect",
	}
	return auditActions[rand.Intn(len(auditActions))]
}

func (wg *WorkloadGenerator) randomRegulatoryAction() string {
	regActions := []string{
		"read",
		"export",
		"verify",
		"audit",
		"inspect",
		"redact",
		"anonymize",
	}
	return regActions[rand.Intn(len(regActions))]
}

func (wg *WorkloadGenerator) randomPurpose() string {
	purposes := []string{
		"analytics",
		"reporting",
		"customer_service",
		"marketing",
		"product_improvement",
		"research",
		"compliance",
	}
	return purposes[rand.Intn(len(purposes))]
}

func (wg *WorkloadGenerator) randomRegulation() string {
	regulations := []string{
		"GDPR",
		"CCPA",
		"HIPAA",
		"PCI-DSS",
		"SOX",
		"GLBA",
		"LGPD",
		"PIPEDA",
	}
	return regulations[rand.Intn(len(regulations))]
}

func (wg *WorkloadGenerator) randomRegulatoryAuthority() string {
	authorities := []string{
		"ICO", // UK
		"CNIL", // France
		"BfDI", // Germany
		"FTC", // US
		"OAIC", // Australia
		"PDPC", // Singapore
		"EDPB", // EU
	}
	return authorities[rand.Intn(len(authorities))]
}

func (wg *WorkloadGenerator) randomDataCategory() string {
	categories := []string{
		"contact_information",
		"financial_details",
		"health_information",
		"biometric_data",
		"behavioral_data",
		"location_data",
		"sensitive_personal_data",
	}
	return categories[rand.Intn(len(categories))]
}

func (wg *WorkloadGenerator) randomTransformationType() string {
	transformations := []string{
		"direct_copy",
		"anonymize",
		"pseudonymize",
		"encrypt",
		"compress",
		"format_conversion",
	}
	return transformations[rand.Intn(len(transformations))]
}
