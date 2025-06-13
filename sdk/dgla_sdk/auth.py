"""
Auth Module - Handles authentication and authorization with DGLA
"""

class AuthModule:
    """Authentication and authorization module for DGLA"""
    
    def __init__(self, client):
        """
        Initialize Auth module
        
        Args:
            client: Reference to the parent DGLAClient
        """
        self.client = client
    
    def login(self, username, password):
        """
        Authenticate with DGLA using credentials
        
        Args:
            username (str): Username
            password (str): Password
            
        Returns:
            dict: Authentication response with token
        """
        response = self.client._request(
            method="POST",
            endpoint="/auth/login",
            json_data={"username": username, "password": password}
        )
        
        # If successful login, update client token
        if "token" in response:
            self.client.token = response["token"]
            self.client.session.headers.update({"Authorization": f"Bearer {response['token']}"})
        
        return response
    
    def verify_token(self):
        """Verify if the current token is valid"""
        return self.client._request("GET", "/auth/verify")
    
    def logout(self):
        """Logout and invalidate token"""
        response = self.client._request("POST", "/auth/logout")
        
        # Clear token on logout
        self.client.token = None
        if "Authorization" in self.client.session.headers:
            del self.client.session.headers["Authorization"]
            
        return response
