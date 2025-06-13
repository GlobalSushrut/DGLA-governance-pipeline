"""
Verify Module - Handles data verification and cryptographic proof functions
"""

class VerifyModule:
    """Client for interacting with DGLA's verification service"""
    
    def __init__(self, client):
        """
        Initialize Verify module
        
        Args:
            client: Reference to the parent DGLAClient
        """
        self.client = client
    
    def verify_data(self, data, hash_algorithm="sha256"):
        """
        Verify data integrity against stored hashes
        
        Args:
            data (dict): Data to verify
            hash_algorithm (str, optional): Hash algorithm to use
            
        Returns:
            dict: Verification result with status and details
        """
        return self.client._request(
            method="POST",
            endpoint="/verify",
            json_data={
                "data": data,
                "algorithm": hash_algorithm
            }
        )
    
    def verify_document(self, document_id, document_hash):
        """
        Verify a document's integrity using its ID and hash
        
        Args:
            document_id (str): Document ID
            document_hash (str): Document hash value
            
        Returns:
            dict: Verification result with status and details
        """
        return self.client._request(
            method="POST",
            endpoint="/verify/document",
            json_data={
                "documentId": document_id,
                "documentHash": document_hash
            }
        )
    
    def create_proof(self, data):
        """
        Create a cryptographic proof for data that can be verified later
        
        Args:
            data (dict): Data to create proof for
            
        Returns:
            dict: Generated proof with ID and hash
        """
        return self.client._request(
            method="POST",
            endpoint="/verify/proof",
            json_data={"data": data}
        )
        
    def create_hash(self, data, algorithm="sha256"):
        """
        Create a cryptographic hash for data
        
        Args:
            data (dict or str): Data to hash
            algorithm (str, optional): Hash algorithm to use
            
        Returns:
            str: Generated hash
        """
        # If this is a dictionary, convert to JSON string first
        if isinstance(data, dict):
            import json
            data_str = json.dumps(data, sort_keys=True)
        else:
            data_str = str(data)
            
        # Calculate the hash locally
        import hashlib
        if algorithm.lower() == "sha256":
            hash_obj = hashlib.sha256(data_str.encode())
        elif algorithm.lower() == "sha512":
            hash_obj = hashlib.sha512(data_str.encode())
        else:
            # Default to SHA-256
            hash_obj = hashlib.sha256(data_str.encode())
            
        return hash_obj.hexdigest()
