"""
Metrics Module - Handles monitoring and metrics reporting
"""

class MetricsModule:
    """Client for interacting with DGLA's metrics and monitoring system"""
    
    def __init__(self, client):
        """
        Initialize Metrics module
        
        Args:
            client: Reference to the parent DGLAClient
        """
        self.client = client
    
    def push_metric(self, metric_name, value, labels=None):
        """
        Push a custom metric to the monitoring system
        
        Args:
            metric_name (str): Name of the metric
            value (float): Value of the metric
            labels (dict, optional): Additional metric labels
            
        Returns:
            dict: Response with status
        """
        payload = {
            "name": metric_name,
            "value": value
        }
        
        if labels:
            payload["labels"] = labels
        
        return self.client._request(
            method="POST",
            endpoint="/metrics/push",
            json_data=payload
        )
    
    def get_metrics(self, metric_names=None):
        """
        Get current metric values
        
        Args:
            metric_names (list, optional): List of metric names to retrieve
            
        Returns:
            dict: Metric values keyed by name
        """
        params = {}
        if metric_names:
            params["names"] = ",".join(metric_names)
            
        return self.client._request(
            method="GET",
            endpoint="/metrics",
            params=params
        )
    
    def create_alert(self, metric_name, threshold, comparison="gt", duration=None, labels=None):
        """
        Create an alert based on metric threshold
        
        Args:
            metric_name (str): Metric to monitor
            threshold (float): Threshold value
            comparison (str): Comparison operator (gt, lt, eq, etc.)
            duration (str, optional): Duration for persistent threshold breach
            labels (dict, optional): Specific labels to match
            
        Returns:
            dict: Created alert details
        """
        payload = {
            "metricName": metric_name,
            "threshold": threshold,
            "comparison": comparison
        }
        
        if duration:
            payload["duration"] = duration
            
        if labels:
            payload["labels"] = labels
        
        return self.client._request(
            method="POST",
            endpoint="/metrics/alerts",
            json_data=payload
        )
    
    def get_prometheus_url(self):
        """
        Get URL of the Prometheus instance
        
        Returns:
            dict: URL information for Prometheus
        """
        return self.client._request(
            method="GET",
            endpoint="/metrics/prometheus-url"
        )
    
    def get_grafana_url(self):
        """
        Get URL of the Grafana instance
        
        Returns:
            dict: URL information for Grafana
        """
        return self.client._request(
            method="GET",
            endpoint="/metrics/grafana-url"
        )
