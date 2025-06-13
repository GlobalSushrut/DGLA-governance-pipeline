"""
Export Module - Handles data export and compliance reporting
"""

class ExportModule:
    """Client for interacting with DGLA's export and compliance reporting service"""
    
    def __init__(self, client):
        """
        Initialize Export module
        
        Args:
            client: Reference to the parent DGLAClient
        """
        self.client = client
    
    def export_logs(self, start_time=None, end_time=None, format_type="json", filters=None):
        """
        Export logs for a specified time period
        
        Args:
            start_time (str, optional): ISO start time for logs
            end_time (str, optional): ISO end time for logs
            format_type (str, optional): Export format (json, csv, pdf)
            filters (dict, optional): Additional filters to apply
            
        Returns:
            dict: Export job details or actual data
        """
        payload = {
            "format": format_type,
        }
        
        if start_time:
            payload["startTime"] = start_time
        
        if end_time:
            payload["endTime"] = end_time
            
        if filters:
            payload["filters"] = filters
        
        return self.client._request(
            method="POST",
            endpoint="/export/logs",
            json_data=payload
        )
    
    def generate_compliance_report(self, report_type, start_time=None, end_time=None, entity_id=None, format=None):
        """
        Generate a compliance report
        
        Args:
            report_type (str): Type of compliance report 
                              (e.g., 'gdpr', 'hipaa', 'pci', 'sox')
            start_time (str, optional): ISO start time for report window
            end_time (str, optional): ISO end time for report window
            entity_id (str, optional): ID of the entity (user, organization, etc.) for the report
            format (str, optional): Format of the report (json, csv, pdf)
            
        Returns:
            dict: Report details or download link
        """
        payload = {
            "reportType": report_type
        }
        
        if start_time:
            payload["startTime"] = start_time
        
        if end_time:
            payload["endTime"] = end_time
            
        if entity_id:
            payload["entityId"] = entity_id
            
        if format:
            payload["format"] = format
        
        return self.client._request(
            method="POST",
            endpoint="/export/compliance",
            json_data=payload
        )
    
    def get_report_status(self, report_id):
        """
        Check status of a previously requested report
        
        Args:
            report_id (str): ID of the report to check
            
        Returns:
            dict: Report status details
        """
        return self.client._request(
            method="GET",
            endpoint=f"/export/status/{report_id}"
        )
    
    def download_report(self, report_id):
        """
        Download a completed report
        
        Args:
            report_id (str): ID of the report to download
            
        Returns:
            bytes: Report data
        """
        response = self.client._request(
            method="GET",
            endpoint=f"/export/download/{report_id}"
        )
        
        return response
