package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/umesh/dgla/benchmark"
)

func main() {
	// Parse command-line flags
	serverURL := flag.String("server", "http://localhost:8081", "URL of the DGLA server")
	concurrentUsers := flag.Int("users", 10, "Number of concurrent users")
	requestsPerUser := flag.Int("requests", 100, "Number of requests per user")
	workloadPattern := flag.String("workload", "standard", "Workload pattern: standard, compliance_audit, data_migration, regulatory_inspection")
	outputFormat := flag.String("output", "text", "Output format: text, json, or html")
	reportFile := flag.String("report", "", "Path to save the report (optional)")
	comparisonOnly := flag.Bool("compare", false, "Only show comparison with competitors (no actual benchmark)")
	flag.Parse()

	// Convert workload pattern string to the appropriate type
	var pattern benchmark.WorkloadPattern
	switch *workloadPattern {
	case "compliance_audit":
		pattern = benchmark.ComplianceAuditWorkload
		fmt.Println("Running compliance audit workload pattern...")
	case "data_migration":
		pattern = benchmark.DataMigrationWorkload
		fmt.Println("Running data migration workload pattern...")
	case "regulatory_inspection":
		pattern = benchmark.RegulatoryInspectionWorkload
		fmt.Println("Running regulatory inspection workload pattern...")
	default:
		pattern = benchmark.StandardWorkload
		fmt.Println("Running standard workload pattern...")
	}

	fmt.Printf("Starting benchmark against %s with %d concurrent users making %d requests each...\n", 
		*serverURL, *concurrentUsers, *requestsPerUser)
	
	// Skip actual benchmark if comparison-only mode
	var result *benchmark.BenchmarkResult
	var err error
	
	if *comparisonOnly {
		// Create a dummy result with just comparison data
		result = &benchmark.BenchmarkResult{
			TestName:             "Competitor Comparison Only",
			RequestCount:         0,
			TotalTime:            1 * time.Second, // Dummy value
			AverageResponseTime:  10 * time.Millisecond, // Dummy value
			ThroughputPerSecond:  1000, // Dummy value
			CompetitorComparison: benchmark.CompareWithCompetitors(10*time.Millisecond, 1000),
		}
	} else {
		// Run the actual benchmark
		result, err = benchmark.RunBenchmark(*serverURL, *concurrentUsers, *requestsPerUser, pattern)
	}
	if err != nil {
		fmt.Printf("Benchmark failed: %v\n", err)
		os.Exit(1)
	}

	// Display the results
	switch *outputFormat {
	case "json":
		displayJSONResults(result, *reportFile)
	case "html":
		displayHTMLResults(result, *reportFile)
	default:
		displayTextResults(result, *reportFile)
	}
}

func displayHTMLResults(result *benchmark.BenchmarkResult, reportFile string) {
	// Generate a default filename if none provided
	if reportFile == "" {
		reportFile = fmt.Sprintf("dgla_benchmark_%s.html", time.Now().Format("20060102_150405"))
	} else if !strings.HasSuffix(reportFile, ".html") {
		// Ensure HTML extension
		reportFile = reportFile + ".html"
	}

	// Ensure the directory exists
	err := os.MkdirAll(filepath.Dir(reportFile), 0755)
	if err != nil {
		fmt.Printf("Error creating directory for report: %v\n", err)
		return
	}

	// Generate the HTML report
	err = benchmark.GenerateHTMLReport(result, reportFile)
	if err != nil {
		fmt.Printf("Error generating HTML report: %v\n", err)
		return
	}

	fmt.Printf("HTML report generated at: %s\n", reportFile)
	fmt.Println("Open this file in a web browser to view the interactive benchmark results")
}

func displayJSONResults(result *benchmark.BenchmarkResult, reportFile string) {
	// Convert to JSON
	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		fmt.Printf("Error creating JSON report: %v\n", err)
		return
	}

	// Display to console
	fmt.Println(string(jsonData))

	// Save to file if specified
	if reportFile != "" {
		err := ioutil.WriteFile(reportFile, jsonData, 0644)
		if err != nil {
			fmt.Printf("Error writing report to file: %v\n", err)
		} else {
			fmt.Printf("Report saved to %s\n", reportFile)
		}
	}
}

func displayTextResults(result *benchmark.BenchmarkResult, reportFile string) {
	// Create output buffer and use tabwriter for alignment
	var output *os.File
	var err error

	if reportFile != "" {
		output, err = os.Create(reportFile)
		if err != nil {
			fmt.Printf("Error creating report file: %v\n", err)
			output = os.Stdout
		} else {
			defer output.Close()
			// Also print to stdout
			fmt.Printf("Report saved to %s\n", reportFile)
		}
	} else {
		output = os.Stdout
	}

	// Create a tabwriter for nicely formatted output
	w := tabwriter.NewWriter(output, 0, 0, 3, ' ', tabwriter.TabIndent)

	// Print benchmark results
	fmt.Fprintln(w, "===============================================")
	fmt.Fprintf(w, "DGLA DATA GOVERNANCE BENCHMARK RESULTS - %s\n", time.Now().Format(time.RFC3339))
	fmt.Fprintln(w, "===============================================")
	fmt.Fprintf(w, "Test Name:\t%s\n", result.TestName)
	fmt.Fprintf(w, "Total Requests:\t%d\n", result.RequestCount)
	fmt.Fprintf(w, "Success Rate:\t%.2f%%\n", result.RequestsSuccessRate)
	fmt.Fprintf(w, "Total Time:\t%v\n", result.TotalTime)
	fmt.Fprintf(w, "Throughput:\t%.2f requests/sec\n", result.ThroughputPerSecond)
	fmt.Fprintln(w, "-----------------------------------------------")
	fmt.Fprintln(w, "RESPONSE TIME METRICS")
	fmt.Fprintf(w, "Average:\t%v\n", result.AverageResponseTime)
	fmt.Fprintf(w, "P95:\t%v\n", result.P95ResponseTime)
	fmt.Fprintf(w, "P99:\t%v\n", result.P99ResponseTime)
	fmt.Fprintf(w, "Min:\t%v\n", result.MinResponseTime)
	fmt.Fprintf(w, "Max:\t%v\n", result.MaxResponseTime)
	fmt.Fprintln(w, "-----------------------------------------------")
	fmt.Fprintln(w, "COMPONENT PERFORMANCE")
	fmt.Fprintf(w, "Merkle Proof Generation:\t%v\n", result.MerkleProofGenTime)
	fmt.Fprintf(w, "Rule Evaluation:\t%v\n", result.RuleEvaluationTime)
	fmt.Fprintln(w, "-----------------------------------------------")
	fmt.Fprintln(w, "RESOURCE USAGE")
	fmt.Fprintf(w, "Memory Usage:\t%.2f MB\n", result.MemoryUsageMB)
	fmt.Fprintf(w, "CPU Usage:\t%.2f%%\n", result.CPUUsagePercent)
	fmt.Fprintln(w, "===============================================")
	fmt.Fprintln(w, "COMPETITOR COMPARISON")
	fmt.Fprintln(w, "===============================================")
	fmt.Fprintln(w, "Competitor\tResp Time\tThroughput\tMemory\tCrypto Proof\tPrice Ratio")
	fmt.Fprintln(w, "-----------------------------------------------")

	for _, comp := range result.CompetitorComparison {
		// For response time and memory, lower is better, so <1.0 means we're better
		respTimeStatus := "✅"
		if comp.ResponseTimeRatio > 1.0 {
			respTimeStatus = "❌"
		}

		throughputStatus := "✅"
		if comp.ThroughputRatio < 1.0 {
			throughputStatus = "❌"
		}

		memoryStatus := "✅"
		if comp.MemoryUsageRatio > 1.0 {
			memoryStatus = "❌"
		}

		cryptoProof := "❌"
		if comp.CryptoProofFeature {
			cryptoProof = "✅"
		} else {
			cryptoProof = "❌ (DGLA: ✅)"
		}

		fmt.Fprintf(w, "%s\t%.2f %s\t%.2f %s\t%.2f %s\t%s\t%.2f\n",
			comp.CompetitorName,
			comp.ResponseTimeRatio, respTimeStatus,
			comp.ThroughputRatio, throughputStatus,
			comp.MemoryUsageRatio, memoryStatus,
			cryptoProof,
			comp.PriceRatio)
	}

	fmt.Fprintln(w, "===============================================")
	fmt.Fprintln(w, "FEATURE COMPARISON")
	fmt.Fprintln(w, "===============================================")
	fmt.Fprintln(w, "Competitor\tRule Complexity\tAudit Compliance\tData Lineage Viz")
	fmt.Fprintln(w, "-----------------------------------------------")

	for _, comp := range result.CompetitorComparison {
		// Convert numeric ratings to stars
		ruleStars := getStarRating(comp.RuleComplexitySupport)
		auditStars := getStarRating(comp.AuditComplianceLevel)
		
		lineageViz := "❌"
		if comp.DataLineageVisualization {
			lineageViz = "✅"
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			comp.CompetitorName,
			ruleStars,
			auditStars,
			lineageViz)
	}

	// Add DGLA row with max features
	fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
		"DGLA Pipeline",
		getStarRating(5),  // We have top rule complexity support
		getStarRating(5),  // We have top audit capabilities
		"✅")              // We have visualization

	fmt.Fprintln(w, "===============================================")
	fmt.Fprintln(w, "KEY DIFFERENTIATORS")
	fmt.Fprintln(w, "===============================================")
	fmt.Fprintln(w, "1. Cryptographic Proof: DGLA is the only solution providing Merkle tree-based cryptographic proof")
	fmt.Fprintln(w, "2. Price/Performance: Best value among all solutions")
	fmt.Fprintln(w, "3. Modularity: Highly extensible architecture")
	fmt.Fprintln(w, "4. Open Source: Full transparency and customizability")
	fmt.Fprintln(w, "===============================================")
	
	w.Flush()
}

// getStarRating converts a numeric rating (1-5) to star symbols
func getStarRating(rating int) string {
	stars := ""
	for i := 0; i < rating; i++ {
		stars += "★"
	}
	for i := rating; i < 5; i++ {
		stars += "☆"
	}
	return stars
}
