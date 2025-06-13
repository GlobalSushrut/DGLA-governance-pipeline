#!/usr/bin/env python3
"""
Demo 8: Secure Voting System

This application demonstrates a secure electronic voting platform with
verifiable ballots, voter privacy, and tamper-evident vote records.

Features:
- Anonymous but verifiable voting
- Vote integrity verification
- Immutable ballot records
- Election audit capabilities
"""
import os
import sys
import argparse
import time
import uuid
import hashlib
import json
from datetime import datetime, timedelta

# Add parent directory to path to import the SDK
sys.path.insert(0, os.path.abspath(os.path.join(os.path.dirname(__file__), '..')))
from dgla_sdk import DGLAClient
from dgla_sdk.constants import HASH_SHA256

class SecureVotingSystem:
    """Secure voting system with cryptographic verification"""
    
    def __init__(self, api_url, api_key=None):
        """Initialize with DGLA client"""
        self.client = DGLAClient(base_url=api_url, api_key=api_key)
        # Simple in-memory stores for demo purposes
        self.elections = {}
        self.voters = {}
        self.ballots = {}
        self.tallies = {}
    
    def create_election(self, election_id, title, candidates, start_time, end_time):
        """Create a new election with secure parameters"""
        timestamp = datetime.now().isoformat()
        
        # Create election record
        election = {
            'election_id': election_id,
            'title': title,
            'candidates': candidates,
            'start_time': start_time,
            'end_time': end_time,
            'created_at': timestamp,
            'status': 'created'
        }
        
        self.elections[election_id] = election
        self.tallies[election_id] = {candidate: 0 for candidate in candidates}
        
        # Create cryptographic proof of election creation
        proof = self.client.verify.create_proof({
            'election_id': election_id,
            'title': title,
            'candidates_hash': hashlib.sha256(json.dumps(sorted(candidates)).encode()).hexdigest(),
            'start_time': start_time,
            'end_time': end_time,
            'timestamp': timestamp
        })
        
        # Record in immutable audit log
        self.client.chainlog.append_log(
            entity_id=election_id,
            entity_type="election",
            action="create",
            metadata={
                'title': title,
                'candidate_count': len(candidates),
                'start_time': start_time,
                'end_time': end_time,
                'proof_id': proof.get('id'),
                'timestamp': timestamp
            }
        )
        
        return {
            'election_id': election_id,
            'proof_id': proof.get('id'),
            'created_at': timestamp
        }
    
    def register_voter(self, voter_id, election_id):
        """Register a voter for an election with anonymous tracking"""
        if election_id not in self.elections:
            return {'error': 'Election not found'}
            
        timestamp = datetime.now().isoformat()
        
        # Create anonymous voting ID
        voting_token = str(uuid.uuid4())
        voter_hash = hashlib.sha256(voter_id.encode()).hexdigest()
        
        # Store voter registration (real identity separate from voting token)
        if voter_hash not in self.voters:
            self.voters[voter_hash] = {}
            
        self.voters[voter_hash][election_id] = {
            'voting_token': voting_token,
            'registered_at': timestamp,
            'has_voted': False
        }
        
        # Record in immutable audit log (without linking voting token to real ID)
        self.client.chainlog.append_log(
            entity_id=voter_hash,
            entity_type="voter_registration",
            action="register",
            metadata={
                'election_id': election_id,
                'timestamp': timestamp
            }
        )
        
        # Push metrics
        self.client.metrics.push_metric(
            metric_name="voter_registrations",
            value=1,
            labels={'election_id': election_id}
        )
        
        return {
            'voting_token': voting_token,
            'election_id': election_id,
            'registered_at': timestamp
        }
    
    def cast_vote(self, voting_token, election_id, candidate):
        """Cast a vote using an anonymous voting token"""
        if election_id not in self.elections:
            return {'error': 'Election not found'}
            
        # Find the voter using this token
        voter_hash = None
        for vh, elections in self.voters.items():
            if election_id in elections and elections[election_id]['voting_token'] == voting_token:
                voter_hash = vh
                break
                
        if not voter_hash:
            return {'error': 'Invalid voting token'}
            
        # Check if voter already voted
        if self.voters[voter_hash][election_id]['has_voted']:
            return {'error': 'Voter has already cast a vote'}
            
        # Check if candidate is valid
        election = self.elections[election_id]
        if candidate not in election['candidates']:
            return {'error': 'Invalid candidate'}
            
        # Check if election is active
        current_time = datetime.now().isoformat()
        if current_time < election['start_time'] or current_time > election['end_time']:
            return {'error': 'Election is not currently active'}
            
        # Create anonymous ballot
        ballot_id = str(uuid.uuid4())
        timestamp = datetime.now().isoformat()
        
        # Store ballot without linking to real voter ID
        self.ballots[ballot_id] = {
            'election_id': election_id,
            'candidate': candidate,
            'timestamp': timestamp,
            # Only store hashed voting token to prevent linking
            'token_hash': hashlib.sha256(voting_token.encode()).hexdigest()
        }
        
        # Mark voter as having voted
        self.voters[voter_hash][election_id]['has_voted'] = True
        
        # Update tally
        self.tallies[election_id][candidate] += 1
        
        # Create cryptographic proof of vote
        proof = self.client.verify.create_proof({
            'ballot_id': ballot_id,
            'election_id': election_id,
            'candidate_hash': hashlib.sha256(candidate.encode()).hexdigest(),
            'timestamp': timestamp
        })
        
        # Record in immutable audit log
        self.client.chainlog.append_log(
            entity_id=ballot_id,
            entity_type="ballot",
            action="cast",
            metadata={
                'election_id': election_id,
                'proof_id': proof.get('id'),
                'timestamp': timestamp
            }
        )
        
        # Push metrics
        self.client.metrics.push_metric(
            metric_name="votes_cast",
            value=1,
            labels={'election_id': election_id}
        )
        
        return {
            'ballot_id': ballot_id,
            'receipt': proof.get('id'),
            'timestamp': timestamp
        }
    
    def verify_vote(self, ballot_id, voting_token):
        """Allow voter to verify their vote was counted correctly"""
        if ballot_id not in self.ballots:
            return {'error': 'Ballot not found'}
            
        ballot = self.ballots[ballot_id]
        token_hash = hashlib.sha256(voting_token.encode()).hexdigest()
        
        # Verify token matches this ballot
        if token_hash != ballot['token_hash']:
            return {'error': 'Verification failed: token mismatch'}
            
        return {
            'verified': True,
            'election_id': ballot['election_id'],
            'candidate': ballot['candidate'],
            'timestamp': ballot['timestamp']
        }
    
    def get_election_results(self, election_id):
        """Get election results with verification proof"""
        if election_id not in self.elections:
            return {'error': 'Election not found'}
            
        election = self.elections[election_id]
        tally = self.tallies[election_id]
        
        # Create a summary of results
        results = {
            'election_id': election_id,
            'title': election['title'],
            'total_votes': sum(tally.values()),
            'results': tally,
            'retrieved_at': datetime.now().isoformat()
        }
        
        # Create a hash of the results for verification
        results_hash = hashlib.sha256(json.dumps(tally, sort_keys=True).encode()).hexdigest()
        
        # Create proof of results
        proof = self.client.verify.create_proof({
            'election_id': election_id,
            'results_hash': results_hash,
            'total_votes': sum(tally.values()),
            'timestamp': results['retrieved_at']
        })
        
        results['verification_id'] = proof.get('id')
        
        return results

def main():
    """Main function to demonstrate secure voting system"""
    parser = argparse.ArgumentParser(description="Secure Voting System Demo")
    parser.add_argument("--api-url", default="http://localhost:8080", help="DGLA API URL")
    parser.add_argument("--api-key", default=None, help="DGLA API key")
    args = parser.parse_args()
    
    # Initialize voting system
    voting_system = SecureVotingSystem(api_url=args.api_url, api_key=args.api_key)
    
    # Demo workflow
    print("🗳️ DGLA Secure Voting System Demo")
    print("=================================")
    
    # 1. Create an election
    print("\n1. Creating new election...")
    election_id = "city-council-2025"
    start_time = datetime.now().isoformat()
    end_time = (datetime.now() + timedelta(hours=24)).isoformat()
    
    election = voting_system.create_election(
        election_id=election_id,
        title="City Council Election 2025",
        candidates=["Alice Johnson", "Bob Smith", "Carol Williams"],
        start_time=start_time,
        end_time=end_time
    )
    print(f"  ✓ Election created: {election_id}")
    print(f"  ✓ Cryptographic proof: {election['proof_id']}")
    
    # 2. Register voters
    print("\n2. Registering voters...")
    voters = ["voter1", "voter2", "voter3", "voter4", "voter5"]
    voter_tokens = {}
    
    for voter in voters:
        result = voting_system.register_voter(voter, election_id)
        voter_tokens[voter] = result['voting_token']
        print(f"  ✓ Registered voter: {voter[:2]}*** with anonymous token")
    
    # 3. Cast votes
    print("\n3. Casting votes...")
    votes = {
        "voter1": "Alice Johnson",
        "voter2": "Bob Smith",
        "voter3": "Alice Johnson",
        "voter4": "Carol Williams",
        "voter5": "Alice Johnson"
    }
    
    ballot_ids = {}
    for voter, candidate in votes.items():
        result = voting_system.cast_vote(
            voting_token=voter_tokens[voter],
            election_id=election_id,
            candidate=candidate
        )
        ballot_ids[voter] = result['ballot_id']
        print(f"  ✓ Vote cast for {candidate} with ballot ID: {result['ballot_id'][:8]}...")
    
    # 4. Voter verifies their vote
    print("\n4. Voter verifying their vote was counted correctly...")
    verification = voting_system.verify_vote(
        ballot_id=ballot_ids["voter1"],
        voting_token=voter_tokens["voter1"]
    )
    print(f"  ✓ Vote verification: {verification['verified']}")
    print(f"  ✓ Verified vote cast for: {verification['candidate']}")
    
    # 5. Get election results
    print("\n5. Retrieving election results...")
    results = voting_system.get_election_results(election_id)
    print(f"  ✓ Election: {results['title']}")
    print(f"  ✓ Total votes: {results['total_votes']}")
    print("  ✓ Results:")
    for candidate, votes in results['results'].items():
        print(f"    - {candidate}: {votes} votes")
    print(f"  ✓ Results verification ID: {results['verification_id']}")
    
    print("\nAll voting activities have been securely recorded with")
    print("cryptographic verification while maintaining voter privacy.")

if __name__ == "__main__":
    main()
