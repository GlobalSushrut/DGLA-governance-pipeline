#!/usr/bin/env python3
"""
Demo 10: Regulatory Compliance Automation

This application demonstrates automatic regulatory compliance monitoring
and reporting across multiple frameworks (GDPR, HIPAA, PCI, SOX, ISO27001).

Features:
- Multi-framework compliance monitoring
- Automated evidence collection
- Continuous compliance verification
- Comprehensive audit reporting
"""
import os
import sys
import argparse
import time
import uuid
import json
from datetime import datetime, timedelta

# Add parent directory to path to import the SDK
sys.path.insert(0, os.path.abspath(os.path.join(os.path.dirname(__file__), '..')))
from dgla_sdk import DGLAClient
from dgla_sdk.constants import (
    FORMAT_JSON, FORMAT_PDF, 
    REPORT_GDPR, REPORT_HIPAA, REPORT_PCI, REPORT_SOX
)

class ComplianceAutomation:
    """Regulatory compliance automation across frameworks"""
    
    def __init__(self, api_url, api_key=None):
        """Initialize with DGLA client"""
        self.client = DGLAClient(base_url=api_url, api_key=api_key)
        # Simple in-memory stores for demo purposes
        self.controls = {}
        self.evidence = {}
        self.assessments = {}
        self.frameworks = {
            "GDPR": {
                "description": "General Data Protection Regulation",
                "control_count": 35
            },
            "HIPAA": {
                "description": "Health Insurance Portability and Accountability Act",
                "control_count": 42
            },
            "PCI-DSS": {
                "description": "Payment Card Industry Data Security Standard",
                "control_count": 28
            },
            "SOX": {
                "description": "Sarbanes-Oxley Act",
                "control_count": 22
            },
            "ISO27001": {
                "description": "International Organization for Standardization security standard",
                "control_count": 46
            }
        }
    
    def register_control(self, control_id, framework, description, impact_level, automated=False):
        """Register a compliance control with the system"""
        timestamp = datetime.now().isoformat()
        
        # Create control record
        control = {
            'control_id': control_id,
            'framework': framework,
            'description': description,
            'impact_level': impact_level,
            'automated': automated,
            'status': 'active',
            'created_at': timestamp
        }
        
        self.controls[control_id] = control
        self.evidence[control_id] = []
        
        # Record in immutable audit log
        self.client.chainlog.append_log(
            entity_id=control_id,
            entity_type="compliance_control",
            action="register",
            metadata={
                'framework': framework,
                'impact_level': impact_level,
                'automated': automated,
                'timestamp': timestamp
            }
        )
        
        return {
            'control_id': control_id,
            'framework': framework,
            'created_at': timestamp
        }
    
    def collect_evidence(self, control_id, evidence_type, source, content):
        """Collect and store evidence for compliance control"""
        if control_id not in self.controls:
            return {'error': 'Control not found'}
            
        timestamp = datetime.now().isoformat()
        evidence_id = str(uuid.uuid4())
        
        # Create evidence record
        evidence = {
            'evidence_id': evidence_id,
            'control_id': control_id,
            'evidence_type': evidence_type,
            'source': source,
            'content': content,
            'timestamp': timestamp
        }
        
        # Add to evidence store
        self.evidence[control_id].append(evidence)
        
        # Create cryptographic proof of evidence
        proof = self.client.verify.create_proof({
            'evidence_id': evidence_id,
            'control_id': control_id,
            'content_hash': self.client.verify.create_hash(content),
            'timestamp': timestamp
        })
        
        # Record in immutable audit log
        self.client.chainlog.append_log(
            entity_id=evidence_id,
            entity_type="compliance_evidence",
            action="collect",
            metadata={
                'control_id': control_id,
                'evidence_type': evidence_type,
                'source': source,
                'proof_id': proof.get('id'),
                'timestamp': timestamp
            }
        )
        
        # Push metrics
        self.client.metrics.push_metric(
            metric_name="evidence_collected",
            value=1,
            labels={
                'framework': self.controls[control_id]['framework'],
                'evidence_type': evidence_type
            }
        )
        
        return {
            'evidence_id': evidence_id,
            'control_id': control_id,
            'proof_id': proof.get('id'),
            'timestamp': timestamp
        }
    
    def assess_control(self, control_id, compliant, assessor=None, notes=None):
        """Assess a control's compliance status"""
        if control_id not in self.controls:
            return {'error': 'Control not found'}
            
        timestamp = datetime.now().isoformat()
        assessment_id = str(uuid.uuid4())
        
        # Create assessment record
        assessment = {
            'assessment_id': assessment_id,
            'control_id': control_id,
            'framework': self.controls[control_id]['framework'],
            'compliant': compliant,
            'assessor': assessor or 'automated-system',
            'notes': notes or '',
            'evidence_count': len(self.evidence[control_id]),
            'timestamp': timestamp
        }
        
        # Store assessment
        if control_id not in self.assessments:
            self.assessments[control_id] = []
            
        self.assessments[control_id].append(assessment)
        
        # Record in immutable audit log
        self.client.chainlog.append_log(
            entity_id=assessment_id,
            entity_type="compliance_assessment",
            action="assess",
            metadata={
                'control_id': control_id,
                'framework': self.controls[control_id]['framework'],
                'compliant': compliant,
                'assessor': assessment['assessor'],
                'evidence_count': assessment['evidence_count'],
                'timestamp': timestamp
            }
        )
        
        # Push metrics
        self.client.metrics.push_metric(
            metric_name="control_assessments",
            value=1,
            labels={
                'framework': self.controls[control_id]['framework'],
                'compliant': str(compliant).lower(),
                'assessor_type': 'automated' if assessor is None else 'manual'
            }
        )
        
        return {
            'assessment_id': assessment_id,
            'control_id': control_id,
            'compliant': compliant,
            'timestamp': timestamp
        }
    
    def get_framework_status(self, framework):
        """Get compliance status for a framework"""
        if framework not in self.frameworks:
            return {'error': 'Framework not found'}
        
        # Find all controls for this framework
        framework_controls = {cid: control for cid, control in self.controls.items() 
                             if control['framework'] == framework}
        
        # Calculate compliance metrics
        total_controls = len(framework_controls)
        assessed_controls = 0
        compliant_controls = 0
        
        for control_id, control in framework_controls.items():
            if control_id in self.assessments and self.assessments[control_id]:
                assessed_controls += 1
                # Get most recent assessment
                latest = sorted(self.assessments[control_id], 
                               key=lambda x: x['timestamp'], 
                               reverse=True)[0]
                if latest['compliant']:
                    compliant_controls += 1
        
        compliance_percentage = (compliant_controls / total_controls * 100) if total_controls > 0 else 0
        
        return {
            'framework': framework,
            'description': self.frameworks[framework]['description'],
            'total_controls': total_controls,
            'assessed_controls': assessed_controls,
            'compliant_controls': compliant_controls,
            'compliance_percentage': round(compliance_percentage, 1),
            'as_of': datetime.now().isoformat()
        }
    
    def generate_audit_report(self, framework, include_evidence=False):
        """Generate comprehensive audit report for compliance framework"""
        if framework not in self.frameworks:
            return {'error': 'Framework not found'}
        
        timestamp = datetime.now().isoformat()
        report_id = str(uuid.uuid4())
        
        # Get framework status
        status = self.get_framework_status(framework)
        
        # Map framework to DGLA report type
        report_type = None
        if framework == "GDPR":
            report_type = REPORT_GDPR
        elif framework == "HIPAA":
            report_type = REPORT_HIPAA
        elif framework == "PCI-DSS":
            report_type = REPORT_PCI
        elif framework == "SOX":
            report_type = REPORT_SOX
        
        # Generate report through DGLA if applicable
        if report_type:
            self.client.export.generate_compliance_report(
                report_type=report_type,
                entity_id=f"framework-{framework}",
                format=FORMAT_PDF
            )
        
        # Record in immutable audit log
        self.client.chainlog.append_log(
            entity_id=report_id,
            entity_type="audit_report",
            action="generate",
            metadata={
                'framework': framework,
                'compliance_percentage': status['compliance_percentage'],
                'include_evidence': include_evidence,
                'timestamp': timestamp
            }
        )
        
        return {
            'report_id': report_id,
            'framework': framework,
            'compliance_percentage': status['compliance_percentage'],
            'generated_at': timestamp,
            'report_url': f"https://reports.example.com/compliance/{framework}/{report_id}"
        }

def main():
    """Main function to demonstrate regulatory compliance automation"""
    parser = argparse.ArgumentParser(description="Regulatory Compliance Automation Demo")
    parser.add_argument("--api-url", default="http://localhost:8080", help="DGLA API URL")
    parser.add_argument("--api-key", default=None, help="DGLA API key")
    args = parser.parse_args()
    
    # Initialize compliance automation
    compliance = ComplianceAutomation(api_url=args.api_url, api_key=args.api_key)
    
    # Demo workflow
    print("📋 DGLA Regulatory Compliance Automation Demo")
    print("============================================")
    
    # 1. Register compliance controls
    print("\n1. Registering compliance controls...")
    
    controls = [
        {
            "id": "gdpr-data-minimization",
            "framework": "GDPR",
            "description": "Ensure data collection is limited to what's necessary",
            "impact": "high",
            "automated": True
        },
        {
            "id": "hipaa-phi-encryption",
            "framework": "HIPAA",
            "description": "Protected Health Information must be encrypted at rest",
            "impact": "critical",
            "automated": True
        },
        {
            "id": "pci-card-storage",
            "framework": "PCI-DSS",
            "description": "Payment card data storage security requirements",
            "impact": "critical", 
            "automated": False
        }
    ]
    
    for control in controls:
        result = compliance.register_control(
            control_id=control["id"],
            framework=control["framework"],
            description=control["description"],
            impact_level=control["impact"],
            automated=control["automated"]
        )
        print(f"  ✓ Registered control: {control['id']} ({control['framework']})")
    
    # 2. Collect compliance evidence
    print("\n2. Collecting compliance evidence...")
    
    # GDPR evidence
    gdpr_evidence = compliance.collect_evidence(
        control_id="gdpr-data-minimization",
        evidence_type="system_config",
        source="database_schema_analyzer",
        content="Database schema verified to only contain necessary fields for business purposes"
    )
    print(f"  ✓ Evidence collected for GDPR control")
    print(f"  ✓ Evidence proof: {gdpr_evidence['proof_id']}")
    
    # HIPAA evidence
    hipaa_evidence = compliance.collect_evidence(
        control_id="hipaa-phi-encryption",
        evidence_type="technical_scan",
        source="encryption_verification_scanner",
        content="All PHI data found to be encrypted with AES-256 at rest"
    )
    print(f"  ✓ Evidence collected for HIPAA control")
    
    # PCI evidence (manual)
    pci_evidence = compliance.collect_evidence(
        control_id="pci-card-storage",
        evidence_type="audit_document",
        source="security_auditor",
        content="Manual audit verified PCI DSS 4.0 requirement 3.1.1 compliance"
    )
    print(f"  ✓ Evidence collected for PCI-DSS control")
    
    # 3. Assess control compliance
    print("\n3. Assessing control compliance...")
    
    # GDPR assessment (automated)
    gdpr_assessment = compliance.assess_control(
        control_id="gdpr-data-minimization",
        compliant=True
    )
    print(f"  ✓ GDPR control assessed: {'Compliant' if gdpr_assessment['compliant'] else 'Non-compliant'}")
    
    # HIPAA assessment (automated)
    hipaa_assessment = compliance.assess_control(
        control_id="hipaa-phi-encryption",
        compliant=True
    )
    print(f"  ✓ HIPAA control assessed: {'Compliant' if hipaa_assessment['compliant'] else 'Non-compliant'}")
    
    # PCI assessment (manual)
    pci_assessment = compliance.assess_control(
        control_id="pci-card-storage",
        compliant=False,
        assessor="John Auditor",
        notes="Card data found in plaintext in legacy database; remediation required"
    )
    print(f"  ✓ PCI-DSS control assessed: {'Compliant' if pci_assessment['compliant'] else 'Non-compliant'}")
    
    # 4. Get framework compliance status
    print("\n4. Getting framework compliance status...")
    
    frameworks = ["GDPR", "HIPAA", "PCI-DSS"]
    for framework in frameworks:
        status = compliance.get_framework_status(framework)
        print(f"  ✓ {framework} compliance: {status['compliance_percentage']}%")
        print(f"  ✓ {status['compliant_controls']} of {status['total_controls']} controls compliant")
    
    # 5. Generate audit reports
    print("\n5. Generating compliance audit reports...")
    
    for framework in frameworks:
        report = compliance.generate_audit_report(framework)
        print(f"  ✓ {framework} audit report generated: {report['report_id']}")
        print(f"  ✓ Report URL: {report['report_url']}")
    
    print("\nAll compliance activities have been cryptographically verified")
    print("and recorded in the immutable audit log to meet regulatory requirements.")

if __name__ == "__main__":
    main()
