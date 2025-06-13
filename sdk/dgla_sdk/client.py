"""
DGLA Client - Main client class for interacting with DGLA services
"""
import requests
import json
import logging
from urllib.parse import urljoin

from .auth import AuthModule
from .chainlog import ChainlogModule
from .verify import VerifyModule
from .export import ExportModule
from .metrics import MetricsModule

logger = logging.getLogger("dgla_sdk")

class DGLAClient:
    """Main client for interacting with DGLA infrastructure"""
    
    def __init__(self, base_url, api_key=None, token=None):
        """
        Initialize DGLA client
        
        Args:
            base_url (str): Base URL of the DGLA API
            api_key (str, optional): API key for authentication
            token (str, optional): JWT token for authentication
        """
        self.base_url = base_url.rstrip('/')
        self.api_key = api_key
        self.token = token
        self.session = requests.Session()
        
        # Add auth headers if provided
        if api_key:
            self.session.headers.update({"X-API-Key": api_key})
        if token:
            self.session.headers.update({"Authorization": f"Bearer {token}"})
        
        # Initialize modules
        self.auth = AuthModule(self)
        self.chainlog = ChainlogModule(self)
        self.verify = VerifyModule(self)
        self.export = ExportModule(self)
        self.metrics = MetricsModule(self)
    
    def _request(self, method, endpoint, params=None, data=None, json_data=None, headers=None):
        """Make a request to the DGLA API"""
        url = urljoin(self.base_url, endpoint)
        req_headers = {}
        if headers:
            req_headers.update(headers)
        
        try:
            response = self.session.request(
                method=method,
                url=url,
                params=params,
                data=data,
                json=json_data,
                headers=req_headers
            )
            
            # Raise for HTTP errors
            response.raise_for_status()
            
            # Return JSON if possible, otherwise return response
            try:
                return response.json()
            except ValueError:
                return response.text
                
        except requests.exceptions.RequestException as e:
            logger.error(f"Error making request to {url}: {e}")
            raise
    
    def healthcheck(self):
        """Check health of DGLA API"""
        return self._request("GET", "/health")
    
    def ready(self):
        """Check if DGLA API is ready"""
        return self._request("GET", "/ready")
    
    def alive(self):
        """Check if DGLA API is alive"""
        return self._request("GET", "/alive")
