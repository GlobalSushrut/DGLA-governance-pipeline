"""
Chainlog Module - Handles immutable logging with blockchain anchoring
"""

class ChainlogModule:
    """Client for interacting with DGLA's immutable chainlog service"""
    
    def __init__(self, client):
        """
        Initialize Chainlog module
        
        Args:
            client: Reference to the parent DGLAClient
        """
        self.client = client
    
    def append_log(self, entity_id, entity_type, action, metadata=None):
        """
        Append a new log entry to the immutable chain
        
        Args:
            entity_id (str): ID of the entity being logged
            entity_type (str): Type of the entity (e.g., 'document', 'user')
            action (str): Action performed (e.g., 'create', 'update', 'delete')
            metadata (dict, optional): Additional metadata for the log entry
            
        Returns:
            dict: Log entry response with ID and hash
        """
        payload = {
            "entityID": entity_id,
            "entityType": entity_type,
            "action": action,
            "metadata": metadata or {}
        }
        
        return self.client._request(
            method="POST",
            endpoint="/logs",
            json_data=payload
        )
    
    def get_logs(self, filters=None, limit=100):
        """
        Get logs from the chain based on filters
        
        Args:
            filters (dict, optional): Filters to apply
            limit (int, optional): Maximum number of logs to return
            
        Returns:
            list: Log entries that match the filters
        """
        params = {"limit": limit}
        if filters:
            params.update(filters)
            
        return self.client._request(
            method="GET",
            endpoint="/logs",
            params=params
        )
    
    def get_log(self, log_id):
        """
        Get a specific log entry by ID
        
        Args:
            log_id (str): ID of the log entry
            
        Returns:
            dict: Log entry details
        """
        return self.client._request(
            method="GET",
            endpoint=f"/logs/{log_id}"
        )
    
    def verify_chain(self):
        """
        Verify the integrity of the entire log chain
        
        Returns:
            dict: Verification result with status and details
        """
        return self.client._request(
            method="POST",
            endpoint="/logs/verify"
        )
