#!/usr/bin/env python3
"""
DGLA IP Protection & Clone Tracking System
------------------------------------------
Monitors and records repository clone events with timestamps and user profiles
to protect intellectual property while maintaining an open source presence.
"""

import os
import sys
import time
import json
import hmac
import hashlib
import logging
import datetime
import requests
from pathlib import Path
from flask import Flask, request, jsonify

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s',
    handlers=[logging.StreamHandler(), logging.FileHandler('ip_tracker.log')]
)
logger = logging.getLogger('dgla-ip-tracker')

# Default paths
TRACKER_DB = Path(__file__).parent / "clone_records.json"
LICENSE_PATH = Path(__file__).parent.parent / "LICENSE.json"
CONFIG_PATH = Path(__file__).parent / "tracker_config.json"
WEBHOOK_SECRET_PATH = Path(__file__).parent / ".webhook_secret"

class CloneTracker:
    """Tracks repository clone events and maintains a database of clone records"""
    
    def __init__(self, db_path=TRACKER_DB, config_path=CONFIG_PATH):
        self.db_path = db_path
        self.config_path = config_path
        
        # Load or create configuration
        self.config = self._load_config()
        
        # Load or create database
        self.db = self._load_db()
        
        # Initialize webhook secret
        self._ensure_webhook_secret()

    def _load_config(self):
        """Load or create configuration"""
        if not Path(self.config_path).exists():
            # Create default config
            config = {
                "repository": "DGLA/data-governance-pipeline",
                "notification_email": "security@dgla-secure.com",
                "alert_threshold": 10,  # Alert after this many clones
                "webhook_url": "https://example.com/api/dgla-ip-tracker",
                "ip_check_enabled": True,
                "automated_responses": {
                    "suspicious_clone": True,
                    "unauthorized_fork": True
                },
                "enable_license_verification": True,
                "enable_clone_watermarking": True
            }
            
            # Save default config
            with open(self.config_path, 'w') as f:
                json.dump(config, f, indent=2)
                
            return config
        else:
            # Load existing config
            try:
                with open(self.config_path, 'r') as f:
                    return json.load(f)
            except json.JSONDecodeError:
                logger.error(f"Invalid configuration file: {self.config_path}")
                return {}

    def _load_db(self):
        """Load or create clone tracking database"""
        if not Path(self.db_path).exists():
            # Create empty database
            db = {
                "clone_events": [],
                "summary": {
                    "total_clones": 0,
                    "unique_users": 0,
                    "last_updated": datetime.datetime.utcnow().isoformat()
                }
            }
            
            # Save empty database
            with open(self.db_path, 'w') as f:
                json.dump(db, f, indent=2)
                
            return db
        else:
            # Load existing database
            try:
                with open(self.db_path, 'r') as f:
                    return json.load(f)
            except json.JSONDecodeError:
                logger.error(f"Invalid database file: {self.db_path}")
                return {
                    "clone_events": [],
                    "summary": {
                        "total_clones": 0,
                        "unique_users": 0,
                        "last_updated": datetime.datetime.utcnow().isoformat()
                    }
                }

    def _ensure_webhook_secret(self):
        """Ensure webhook secret exists"""
        if not Path(WEBHOOK_SECRET_PATH).exists():
            # Generate a new secret
            secret = hashlib.sha256(os.urandom(32)).hexdigest()
            
            # Save secret
            with open(WEBHOOK_SECRET_PATH, 'w') as f:
                f.write(secret)
                
            logger.info(f"Generated new webhook secret: {WEBHOOK_SECRET_PATH}")
            
    def record_clone_event(self, user_profile, ip_address=None, user_agent=None):
        """Record a repository clone event"""
        timestamp = datetime.datetime.utcnow().isoformat()
        
        # Create clone record
        clone_event = {
            "event_id": hashlib.sha256(f"{user_profile}:{timestamp}".encode()).hexdigest()[:16],
            "timestamp": timestamp,
            "user_profile": user_profile,
            "clone_type": "https" if not ip_address else "git",
            "ip_metadata": {
                "address": ip_address,
                "geo_location": self._get_geo_location(ip_address) if ip_address else None
            },
            "user_agent": user_agent,
            "verified_license_agreement": self._check_license_requirement(),
            "watermark_applied": self._apply_clone_watermark(user_profile, timestamp) if self.config.get("enable_clone_watermarking") else False
        }
        
        # Add to database
        self.db["clone_events"].append(clone_event)
        self.db["summary"]["total_clones"] += 1
        
        # Update unique users count
        user_profiles = set([event["user_profile"] for event in self.db["clone_events"]])
        self.db["summary"]["unique_users"] = len(user_profiles)
        self.db["summary"]["last_updated"] = timestamp
        
        # Save database
        self._save_db()
        
        # Check if alert threshold reached
        if self._check_alert_threshold(user_profile):
            self._send_alert(user_profile)
            
        logger.info(f"Recorded clone event: {user_profile} at {timestamp} (ID: {clone_event['event_id']})")
        
        return clone_event
        
    def _get_geo_location(self, ip_address):
        """Get geolocation information for an IP address"""
        # In a production environment, this would call a geolocation API
        # For demo purposes, we'll return placeholder data
        return {
            "country": "Unknown",
            "region": "Unknown",
            "city": "Unknown",
            "coordinates": [0, 0],
            "isp": "Unknown"
        }
        
    def _check_license_requirement(self):
        """Check if license requirements have been met"""
        # In production, this would verify license acceptance
        # For demo purposes, we'll return True
        return True
        
    def _apply_clone_watermark(self, user_profile, timestamp):
        """Apply a watermark to the cloned repository"""
        # In production, this would inject a unique watermark into the clone
        # For demo, we'll just return True to indicate watermarking is enabled
        return True
        
    def _save_db(self):
        """Save clone tracking database"""
        with open(self.db_path, 'w') as f:
            json.dump(self.db, f, indent=2)
            
    def _check_alert_threshold(self, user_profile):
        """Check if alert threshold has been reached for a user"""
        # Count clones by this user
        user_clones = [event for event in self.db["clone_events"] 
                        if event["user_profile"] == user_profile]
        
        return len(user_clones) >= self.config.get("alert_threshold", 10)
        
    def _send_alert(self, user_profile):
        """Send alert about suspicious clone activity"""
        # In production, this would send an email or notification
        logger.warning(f"Alert threshold reached for user: {user_profile}")
        
    def get_clone_statistics(self):
        """Get clone statistics summary"""
        return {
            "total_clones": self.db["summary"]["total_clones"],
            "unique_users": self.db["summary"]["unique_users"],
            "last_updated": self.db["summary"]["last_updated"],
            "clones_last_24h": len([event for event in self.db["clone_events"] 
                                    if (datetime.datetime.fromisoformat(event["timestamp"]) > 
                                        datetime.datetime.utcnow() - datetime.timedelta(days=1))])
        }
        
    def verify_webhook_signature(self, signature, payload):
        """Verify GitHub webhook signature"""
        try:
            with open(WEBHOOK_SECRET_PATH, 'r') as f:
                secret = f.read().strip()
                
            expected = 'sha256=' + hmac.new(
                secret.encode(), 
                payload.encode(), 
                hashlib.sha256
            ).hexdigest()
            
            return hmac.compare_digest(signature, expected)
        except Exception as e:
            logger.error(f"Webhook signature verification failed: {str(e)}")
            return False


# Create Flask app for webhook handling
app = Flask(__name__)
tracker = CloneTracker()

@app.route('/webhook/github', methods=['POST'])
def github_webhook():
    """Handle GitHub webhooks for clone events"""
    # Verify signature
    signature = request.headers.get('X-Hub-Signature-256')
    if not signature or not tracker.verify_webhook_signature(signature, request.get_data(as_text=True)):
        logger.warning("Invalid webhook signature")
        return jsonify({"status": "error", "message": "Invalid signature"}), 401
    
    # Process webhook
    event_type = request.headers.get('X-GitHub-Event')
    payload = request.json
    
    if event_type == 'clone':
        # Record clone event
        user_profile = payload.get('sender', {}).get('login', 'anonymous')
        ip_address = payload.get('client_ip')
        user_agent = payload.get('client_user_agent')
        
        event = tracker.record_clone_event(user_profile, ip_address, user_agent)
        return jsonify({"status": "success", "event_id": event["event_id"]})
        
    return jsonify({"status": "ignored", "message": "Unhandled event type"})

@app.route('/stats', methods=['GET'])
def stats():
    """Get clone statistics"""
    return jsonify(tracker.get_clone_statistics())

@app.route('/health', methods=['GET'])
def health():
    """Health check endpoint"""
    return jsonify({"status": "healthy", "timestamp": datetime.datetime.utcnow().isoformat()})

def setup_github_webhook(repo, url, secret):
    """Set up GitHub webhook for clone events"""
    # This would need to use the GitHub API to create a webhook
    # For demo purposes, we'll just print instructions
    logger.info(f"To set up the GitHub webhook:")
    logger.info(f"1. Go to your repository settings: https://github.com/{repo}/settings/hooks")
    logger.info(f"2. Add webhook with URL: {url}")
    logger.info(f"3. Set content type to application/json")
    logger.info(f"4. Set secret to the value in {WEBHOOK_SECRET_PATH}")
    logger.info(f"5. Select 'Let me select individual events' and check 'Clone' and 'Fork'")
    
def main():
    """Main entry point for the IP tracker"""
    if len(sys.argv) > 1 and sys.argv[1] == '--setup-webhook':
        # Set up webhook
        with open(CONFIG_PATH, 'r') as f:
            config = json.load(f)
            
        with open(WEBHOOK_SECRET_PATH, 'r') as f:
            secret = f.read().strip()
            
        setup_github_webhook(
            config.get("repository", "DGLA/data-governance-pipeline"),
            config.get("webhook_url", "https://example.com/api/dgla-ip-tracker"),
            secret
        )
    elif len(sys.argv) > 1 and sys.argv[1] == '--server':
        # Start webhook server
        app.run(host='0.0.0.0', port=5000)
    else:
        # Print statistics
        stats = tracker.get_clone_statistics()
        print(f"\nDGLA IP TRACKER STATISTICS")
        print(f"==========================")
        print(f"Total Clones: {stats['total_clones']}")
        print(f"Unique Users: {stats['unique_users']}")
        print(f"Clones (24h): {stats['clones_last_24h']}")
        print(f"Last Updated: {stats['last_updated']}")
        print("\nUse --setup-webhook to configure GitHub webhook")
        print("Use --server to start the webhook server")

if __name__ == "__main__":
    main()
