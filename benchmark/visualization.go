package benchmark

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"
)

// ChartType defines the type of visualization
type ChartType string

const (
	// BarChart represents a bar chart visualization
	BarChart ChartType = "bar"
	
	// LineChart represents a line chart visualization
	LineChart ChartType = "line"
	
	// RadarChart represents a radar/spider chart for multiple metrics
	RadarChart ChartType = "radar"
)

// generateHTMLReport creates an HTML report with interactive visualizations
func GenerateHTMLReport(result *BenchmarkResult, outputFile string) error {
	// Create HTML content with visualizations
	html := generateHTMLContent(result)
	
	// Write to file
	return ioutil.WriteFile(outputFile, []byte(html), 0644)
}

// generateHTMLContent creates the HTML content with visualizations
func generateHTMLContent(result *BenchmarkResult) string {
	var buffer bytes.Buffer
	
	// Write HTML header
	buffer.WriteString(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>DGLA Pipeline Benchmark Report</title>
    <script src="https://cdn.jsdelivr.net/npm/chart.js"></script>
    <script src="https://cdn.jsdelivr.net/npm/chartjs-chart-matrix@1.2.0/dist/chartjs-chart-matrix.min.js"></script>
    <script src="https://cdn.jsdelivr.net/npm/chartjs-plugin-datalabels@2.0.0"></script>
    <style>
        body {
            font-family: Arial, sans-serif;
            margin: 20px;
            background-color: #f5f5f5;
        }
        .container {
            max-width: 1200px;
            margin: 0 auto;
            background-color: white;
            padding: 20px;
            border-radius: 8px;
            box-shadow: 0 0 10px rgba(0,0,0,0.1);
        }
        .header {
            text-align: center;
            padding: 20px 0;
            border-bottom: 1px solid #eee;
            margin-bottom: 30px;
        }
        .summary {
            display: flex;
            justify-content: space-between;
            flex-wrap: wrap;
            margin-bottom: 30px;
        }
        .metric-card {
            background-color: #f9f9f9;
            border-radius: 8px;
            padding: 15px;
            width: 23%;
            box-sizing: border-box;
            text-align: center;
            box-shadow: 0 0 5px rgba(0,0,0,0.05);
        }
        .metric-value {
            font-size: 24px;
            font-weight: bold;
            color: #2c3e50;
        }
        .metric-label {
            font-size: 14px;
            color: #7f8c8d;
        }
        .chart-container {
            margin: 40px 0;
        }
        .chart-title {
            font-size: 18px;
            margin-bottom: 15px;
            color: #34495e;
        }
        .chart {
            height: 400px;
        }
        .feature-table {
            width: 100%;
            border-collapse: collapse;
            margin: 30px 0;
        }
        .feature-table th, .feature-table td {
            border: 1px solid #ddd;
            padding: 12px;
            text-align: left;
        }
        .feature-table th {
            background-color: #f2f2f2;
        }
        .feature-table tr:nth-child(even) {
            background-color: #f9f9f9;
        }
        .better {
            color: #27ae60;
            font-weight: bold;
        }
        .worse {
            color: #e74c3c;
            font-weight: bold;
        }
        .footer {
            text-align: center;
            padding: 20px 0;
            margin-top: 30px;
            border-top: 1px solid #eee;
            color: #7f8c8d;
            font-size: 14px;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>DGLA Pipeline Benchmark Report</h1>
            <p>Generated on `)
	
	// Add generation time
	buffer.WriteString(time.Now().Format(time.RFC1123))
	buffer.WriteString(`</p>
        </div>

        <div class="summary">
            <div class="metric-card">
                <div class="metric-value">`)
	buffer.WriteString(fmt.Sprintf("%d", result.RequestCount))
	buffer.WriteString(`</div>
                <div class="metric-label">Total Requests</div>
            </div>
            <div class="metric-card">
                <div class="metric-value">`)
	buffer.WriteString(fmt.Sprintf("%.2f req/s", result.ThroughputPerSecond))
	buffer.WriteString(`</div>
                <div class="metric-label">Throughput</div>
            </div>
            <div class="metric-card">
                <div class="metric-value">`)
	buffer.WriteString(fmt.Sprintf("%.2f ms", float64(result.AverageResponseTime.Milliseconds())))
	buffer.WriteString(`</div>
                <div class="metric-label">Avg Response Time</div>
            </div>
            <div class="metric-card">
                <div class="metric-value">`)
	buffer.WriteString(fmt.Sprintf("%.1f%%", result.RequestsSuccessRate))
	buffer.WriteString(`</div>
                <div class="metric-label">Success Rate</div>
            </div>
        </div>

        <div class="chart-container">
            <h2 class="chart-title">Response Time Comparison (ms)</h2>
            <canvas id="responseTimeChart" class="chart"></canvas>
        </div>

        <div class="chart-container">
            <h2 class="chart-title">Feature Comparison</h2>
            <canvas id="featureComparisonChart" class="chart"></canvas>
        </div>

        <div class="chart-container">
            <h2 class="chart-title">Performance Radar</h2>
            <canvas id="performanceRadarChart" class="chart"></canvas>
        </div>

        <h2 class="chart-title">Detailed Feature Comparison</h2>
        <table class="feature-table">
            <thead>
                <tr>
                    <th>Feature</th>
                    <th>DGLA Pipeline</th>`)
	
	// Add competitor names as table headers
	for compName := range result.CompetitorComparison {
		buffer.WriteString(fmt.Sprintf("<th>%s</th>", compName))
	}
	
	buffer.WriteString(`
                </tr>
            </thead>
            <tbody>
                <tr>
                    <td>Cryptographic Proof</td>
                    <td>✅</td>`)
	
	// Add crypto proof support for competitors
	for _, comp := range result.CompetitorComparison {
		if comp.CryptoProofFeature {
			buffer.WriteString("<td>✅</td>")
		} else {
			buffer.WriteString("<td>❌</td>")
		}
	}
	
	buffer.WriteString(`
                </tr>
                <tr>
                    <td>Rule Complexity</td>
                    <td>★★★★★</td>`)
	
	// Add rule complexity for competitors
	for _, comp := range result.CompetitorComparison {
		stars := strings.Repeat("★", comp.RuleComplexitySupport) + 
		          strings.Repeat("☆", 5-comp.RuleComplexitySupport)
		buffer.WriteString(fmt.Sprintf("<td>%s</td>", stars))
	}
	
	buffer.WriteString(`
                </tr>
                <tr>
                    <td>Audit Compliance</td>
                    <td>★★★★★</td>`)
	
	// Add audit compliance for competitors
	for _, comp := range result.CompetitorComparison {
		stars := strings.Repeat("★", comp.AuditComplianceLevel) + 
		          strings.Repeat("☆", 5-comp.AuditComplianceLevel)
		buffer.WriteString(fmt.Sprintf("<td>%s</td>", stars))
	}
	
	buffer.WriteString(`
                </tr>
                <tr>
                    <td>Data Lineage Visualization</td>
                    <td>✅</td>`)
	
	// Add lineage visualization for competitors
	for _, comp := range result.CompetitorComparison {
		if comp.DataLineageVisualization {
			buffer.WriteString("<td>✅</td>")
		} else {
			buffer.WriteString("<td>❌</td>")
		}
	}
	
	buffer.WriteString(`
                </tr>
                <tr>
                    <td>Relative Cost</td>
                    <td>1.0x</td>`)
	
	// Add price ratio for competitors
	for _, comp := range result.CompetitorComparison {
		buffer.WriteString(fmt.Sprintf("<td>%.1fx</td>", 1/comp.PriceRatio))
	}
	
	buffer.WriteString(`
                </tr>
            </tbody>
        </table>

        <div class="footer">
            <p>DGLA Pipeline - Enterprise-grade Data Governance and Lineage Tracking System</p>
        </div>
    </div>

    <script>
        // Response Time Chart
        const responseTimeCtx = document.getElementById('responseTimeChart').getContext('2d');
        const responseTimeChart = new Chart(responseTimeCtx, {
            type: 'bar',
            data: {
                labels: ['DGLA Pipeline'`)
	
	// Add competitor names to labels
	for compName := range result.CompetitorComparison {
		buffer.WriteString(fmt.Sprintf(", '%s'", compName))
	}
	
	buffer.WriteString(`],
                datasets: [{
                    label: 'Average Response Time (ms)',
                    data: [`)
	
	// Add DGLA response time
	buffer.WriteString(fmt.Sprintf("%.2f", float64(result.AverageResponseTime.Milliseconds())))
	
	// Add competitor response times
	for _, comp := range result.CompetitorComparison {
		competitorRespTime := float64(result.AverageResponseTime.Milliseconds()) / comp.ResponseTimeRatio
		buffer.WriteString(fmt.Sprintf(", %.2f", competitorRespTime))
	}
	
	buffer.WriteString(`],
                    backgroundColor: [
                        'rgba(54, 162, 235, 0.7)',`)
	
	// Add colors for competitors
	buffer.WriteString(strings.Repeat("'rgba(255, 99, 132, 0.7)',", len(result.CompetitorComparison)))
	
	buffer.WriteString(`
                    ],
                    borderColor: [
                        'rgba(54, 162, 235, 1)',`)
	
	// Add border colors for competitors
	buffer.WriteString(strings.Repeat("'rgba(255, 99, 132, 1)',", len(result.CompetitorComparison)))
	
	buffer.WriteString(`
                    ],
                    borderWidth: 1
                }]
            },
            options: {
                scales: {
                    y: {
                        beginAtZero: true,
                        title: {
                            display: true,
                            text: 'Response Time (ms)'
                        }
                    }
                },
                plugins: {
                    title: {
                        display: true,
                        text: 'Average Response Time Comparison'
                    }
                }
            }
        });

        // Feature Comparison Chart
        const featureComparisonCtx = document.getElementById('featureComparisonChart').getContext('2d');
        const featureComparisonChart = new Chart(featureComparisonCtx, {
            type: 'bar',
            data: {
                labels: ['Cryptographic Proof', 'Rule Complexity', 'Audit Compliance', 'Data Lineage', 'Cost Efficiency'],
                datasets: [{
                    label: 'DGLA Pipeline',
                    data: [100, 100, 100, 100, 100],
                    backgroundColor: 'rgba(54, 162, 235, 0.7)',
                    borderColor: 'rgba(54, 162, 235, 1)',
                    borderWidth: 1
                },`)
	
	// Add datasets for competitors
	compIndex := 0
	for compName, comp := range result.CompetitorComparison {
		buffer.WriteString(fmt.Sprintf(`
                {
                    label: '%s',
                    data: [`, compName))
		
		// Crypto proof
		if comp.CryptoProofFeature {
			buffer.WriteString("100")
		} else {
			buffer.WriteString("0")
		}
		
		// Rule complexity (as percentage of 5 stars)
		buffer.WriteString(fmt.Sprintf(", %d", comp.RuleComplexitySupport*20))
		
		// Audit compliance (as percentage of 5 stars)
		buffer.WriteString(fmt.Sprintf(", %d", comp.AuditComplianceLevel*20))
		
		// Data lineage
		if comp.DataLineageVisualization {
			buffer.WriteString(", 100")
		} else {
			buffer.WriteString(", 0")
		}
		
		// Cost efficiency (inverse of price ratio, as percentage)
		costEfficiency := int((1 / comp.PriceRatio) * 100)
		if costEfficiency > 100 {
			costEfficiency = 100
		}
		buffer.WriteString(fmt.Sprintf(", %d", costEfficiency))
		
		// Assign different colors based on competitor index
		var bgColor, borderColor string
		switch compIndex % 3 {
		case 1:
			bgColor = "'rgba(75, 192, 192, 0.7)'"
			borderColor = "'rgba(75, 192, 192, 1)'"
		case 2:
			bgColor = "'rgba(153, 102, 255, 0.7)'"
			borderColor = "'rgba(153, 102, 255, 1)'"
		default:
			bgColor = "'rgba(255, 99, 132, 0.7)'"
			borderColor = "'rgba(255, 99, 132, 1)'"
		}
		
		buffer.WriteString(`],
                    backgroundColor: `)
		buffer.WriteString(bgColor)
		buffer.WriteString(`,
                    borderColor: `)
		buffer.WriteString(borderColor)
		
		buffer.WriteString(`,
                    borderWidth: 1
                },`)
		
		compIndex++
	}
	
	// Remove the trailing comma from the last dataset
	if len(result.CompetitorComparison) > 0 {
		buffer.Truncate(buffer.Len() - 1)
	}
	
	buffer.WriteString(`
                ]
            },
            options: {
                scales: {
                    y: {
                        beginAtZero: true,
                        max: 100,
                        title: {
                            display: true,
                            text: 'Feature Score (%)'
                        }
                    }
                },
                plugins: {
                    title: {
                        display: true,
                        text: 'Feature Comparison'
                    }
                }
            }
        });

        // Performance Radar Chart
        const performanceRadarCtx = document.getElementById('performanceRadarChart').getContext('2d');
        const performanceRadarChart = new Chart(performanceRadarCtx, {
            type: 'radar',
            data: {
                labels: ['Response Time', 'Throughput', 'Memory Usage', 'Feature Richness', 'Cost Efficiency'],
                datasets: [{
                    label: 'DGLA Pipeline',
                    data: [100, 100, 100, 100, 100],
                    backgroundColor: 'rgba(54, 162, 235, 0.2)',
                    borderColor: 'rgba(54, 162, 235, 1)',
                    borderWidth: 2,
                    pointBackgroundColor: 'rgba(54, 162, 235, 1)'
                },`)
	
	// Add datasets for competitors in radar chart
	compIndex = 0
	for compName, comp := range result.CompetitorComparison {
		buffer.WriteString(fmt.Sprintf(`
                {
                    label: '%s',
                    data: [`, compName))
		
		// Response time (lower is better, so invert ratio)
		respTimeScore := int((1 / comp.ResponseTimeRatio) * 100)
		if respTimeScore > 100 {
			respTimeScore = 100
		}
		buffer.WriteString(fmt.Sprintf("%d", respTimeScore))
		
		// Throughput (higher is better)
		throughputScore := int(comp.ThroughputRatio * 100)
		if throughputScore > 100 {
			throughputScore = 100
		}
		buffer.WriteString(fmt.Sprintf(", %d", throughputScore))
		
		// Memory usage (lower is better, so invert ratio)
		memoryScore := int((1 / comp.MemoryUsageRatio) * 100)
		if memoryScore > 100 {
			memoryScore = 100
		}
		buffer.WriteString(fmt.Sprintf(", %d", memoryScore))
		
		// Feature richness (average of rule complexity and audit compliance)
		featureScore := (comp.RuleComplexitySupport + comp.AuditComplianceLevel) * 10
		buffer.WriteString(fmt.Sprintf(", %d", featureScore))
		
		// Cost efficiency (inverse of price ratio, as percentage)
		costEfficiency := int((1 / comp.PriceRatio) * 100)
		if costEfficiency > 100 {
			costEfficiency = 100
		}
		buffer.WriteString(fmt.Sprintf(", %d", costEfficiency))
		
		// Assign different colors based on competitor index for radar chart
		var radarBgColor, radarBorderColor, radarPointColor string
		switch compIndex % 3 {
		case 1:
			radarBgColor = "'rgba(75, 192, 192, 0.2)'"
			radarBorderColor = "'rgba(75, 192, 192, 1)'"
			radarPointColor = "'rgba(75, 192, 192, 1)'"
		case 2:
			radarBgColor = "'rgba(153, 102, 255, 0.2)'"
			radarBorderColor = "'rgba(153, 102, 255, 1)'"
			radarPointColor = "'rgba(153, 102, 255, 1)'"
		default:
			radarBgColor = "'rgba(255, 99, 132, 0.2)'"
			radarBorderColor = "'rgba(255, 99, 132, 1)'"
			radarPointColor = "'rgba(255, 99, 132, 1)'"
		}
		
		buffer.WriteString(`],
                    backgroundColor: `)
		buffer.WriteString(radarBgColor)
		buffer.WriteString(`,
                    borderColor: `)
		buffer.WriteString(radarBorderColor)
		buffer.WriteString(`,
                    borderWidth: 2,
                    pointBackgroundColor: `)
		buffer.WriteString(radarPointColor)
		
		buffer.WriteString(`
                },`)
		
		compIndex++
	}
	
	// Remove the trailing comma from the last dataset
	if len(result.CompetitorComparison) > 0 {
		buffer.Truncate(buffer.Len() - 1)
	}
	
	buffer.WriteString(`
                ]
            },
            options: {
                elements: {
                    line: {
                        tension: 0.1
                    }
                },
                scales: {
                    r: {
                        angleLines: {
                            display: true
                        },
                        suggestedMin: 0,
                        suggestedMax: 100
                    }
                },
                plugins: {
                    title: {
                        display: true,
                        text: 'Performance Radar Comparison'
                    }
                }
            }
        });
    </script>
</body>
</html>`)

	return buffer.String()
}

// saveChart saves a chart as an image file
func saveChart(chartType ChartType, title string, labels []string, datasets []map[string]interface{}, outputFile string) error {
	// This would typically use a chart rendering library
	// For simplicity in this example, we'll just create a placeholder file
	
	f, err := os.Create(outputFile)
	if err != nil {
		return err
	}
	defer f.Close()
	
	f.WriteString(fmt.Sprintf("Chart: %s\n", title))
	f.WriteString(fmt.Sprintf("Type: %s\n", chartType))
	f.WriteString("Labels: ")
	for _, label := range labels {
		f.WriteString(label + ", ")
	}
	f.WriteString("\n")
	
	return nil
}
