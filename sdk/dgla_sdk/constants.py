"""
Constants for the DGLA SDK
"""

# API Endpoints
ENDPOINT_HEALTH = "/health"
ENDPOINT_READY = "/ready"
ENDPOINT_ALIVE = "/alive"
ENDPOINT_AUTH = "/auth"
ENDPOINT_LOGS = "/logs"
ENDPOINT_VERIFY = "/verify"
ENDPOINT_EXPORT = "/export"
ENDPOINT_METRICS = "/metrics"

# Verification algorithms
HASH_SHA256 = "sha256"
HASH_SHA512 = "sha512"
HASH_BLAKE2B = "blake2b"

# Export formats
FORMAT_JSON = "json"
FORMAT_CSV = "csv"
FORMAT_PDF = "pdf"

# Compliance report types
COMPLIANCE_GDPR = "gdpr"
COMPLIANCE_HIPAA = "hipaa"
COMPLIANCE_PCI = "pci"
COMPLIANCE_SOX = "sox"
COMPLIANCE_ISO27001 = "iso27001"

# Report types
REPORT_GDPR = "gdpr"
REPORT_HIPAA = "hipaa"
REPORT_PCI = "pci"
REPORT_SOX = "sox"
REPORT_ISO27001 = "iso27001"

# Metric comparison operators
COMPARISON_GT = "gt"  # Greater than
COMPARISON_LT = "lt"  # Less than
COMPARISON_EQ = "eq"  # Equal to
COMPARISON_GTE = "gte"  # Greater than or equal to
COMPARISON_LTE = "lte"  # Less than or equal to
