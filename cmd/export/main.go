package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/jung-kurt/gofpdf"
)

const (
	version = "1.0.0"
)

// AuditRecord represents a single audit record
type AuditRecord struct {
	JobID      string    `json:"job_id"`
	Timestamp  time.Time `json:"timestamp"`
	RuleHash   string    `json:"rule_hash"`
	MerkleLeaf string    `json:"merkle_leaf"`
	ProofID    string    `json:"proof_id"`
}

// AuditExport represents a collection of audit records
type AuditExport struct {
	Records    []AuditRecord `json:"records"`
	ExportDate time.Time     `json:"export_date"`
	ExportID   string        `json:"export_id"`
}

func main() {
	// Define command-line flags
	inputFilePtr := flag.String("input", "", "Path to input JSON file")
	outputFilePtr := flag.String("output", "", "Path to output file (will determine format based on extension)")
	formatPtr := flag.String("format", "json", "Output format (json or pdf)")
	versionPtr := flag.Bool("version", false, "Print version information")

	flag.Parse()

	// Print version if requested
	if *versionPtr {
		fmt.Printf("DGLA Audit Export Tool v%s\n", version)
		return
	}

	// Check required parameters
	if *inputFilePtr == "" {
		log.Fatal("Error: input file is required")
	}

	if *outputFilePtr == "" {
		log.Fatal("Error: output file is required")
	}

	// Load audit data
	auditData, err := loadAuditData(*inputFilePtr)
	if err != nil {
		log.Fatalf("Error: failed to load audit data: %v", err)
	}

	// Determine output format
	format := *formatPtr
	if format == "" {
		// Try to determine from file extension
		ext := filepath.Ext(*outputFilePtr)
		switch ext {
		case ".json":
			format = "json"
		case ".pdf":
			format = "pdf"
		default:
			log.Fatalf("Error: unknown output format: %s", ext)
		}
	}

	// Export data based on format
	switch format {
	case "json":
		if err := exportToJSON(auditData, *outputFilePtr); err != nil {
			log.Fatalf("Error: failed to export to JSON: %v", err)
		}
	case "pdf":
		if err := exportToPDF(auditData, *outputFilePtr); err != nil {
			log.Fatalf("Error: failed to export to PDF: %v", err)
		}
	default:
		log.Fatalf("Error: unsupported format: %s", format)
	}

	fmt.Printf("Audit data exported to %s in %s format\n", *outputFilePtr, format)
}

// Load audit data from file
func loadAuditData(path string) (*AuditExport, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %v", err)
	}

	var auditData AuditExport
	if err := json.Unmarshal(data, &auditData); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %v", err)
	}

	return &auditData, nil
}

// Export audit data to JSON
func exportToJSON(auditData *AuditExport, path string) error {
	data, err := json.MarshalIndent(auditData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %v", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %v", err)
	}

	return nil
}

// Export audit data to PDF
func exportToPDF(auditData *AuditExport, path string) error {
	// Create a new PDF
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()

	// Set font
	pdf.SetFont("Arial", "B", 16)

	// Add title
	pdf.Cell(40, 10, "DGLA Audit Export")
	pdf.Ln(10)

	// Add export metadata
	pdf.SetFont("Arial", "", 12)
	pdf.Cell(40, 10, fmt.Sprintf("Export Date: %s", auditData.ExportDate.Format("2006-01-02 15:04:05")))
	pdf.Ln(8)
	pdf.Cell(40, 10, fmt.Sprintf("Export ID: %s", auditData.ExportID))
	pdf.Ln(15)

	// Add table header
	pdf.SetFont("Arial", "B", 12)
	pdf.Cell(40, 10, "Job ID")
	pdf.Cell(40, 10, "Timestamp")
	pdf.Cell(40, 10, "Rule Hash")
	pdf.Cell(40, 10, "Proof ID")
	pdf.Ln(10)

	// Add table rows
	pdf.SetFont("Arial", "", 10)
	for _, record := range auditData.Records {
		pdf.Cell(40, 10, record.JobID)
		pdf.Cell(40, 10, record.Timestamp.Format("2006-01-02 15:04:05"))
		pdf.Cell(40, 10, truncateString(record.RuleHash, 20))
		pdf.Cell(40, 10, truncateString(record.ProofID, 20))
		pdf.Ln(8)
	}

	// Add footer
	pdf.SetY(-15)
	pdf.SetFont("Arial", "I", 8)
	pdf.Cell(0, 10, fmt.Sprintf("DGLA Audit Export - Page %d/{nb}", pdf.PageNo()))

	// Save the PDF
	return pdf.OutputFileAndClose(path)
}

// Helper function to truncate long strings
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
