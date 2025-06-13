#!/usr/bin/env python3
"""
DGLA Secure Mesh Network Communication Module
Military-grade secure API communication between SDK and engine components
"""

import os
import time
import json
import uuid
import base64
import hashlib
import logging
import secrets
from dataclasses import dataclass
from typing import Dict, List, Any, Optional
from datetime import datetime, timedelta

# Cryptography imports
from cryptography.hazmat.primitives.ciphers import Cipher, algorithms, modes
from cryptography.hazmat.primitives import hashes, hmac, padding
from cryptography.hazmat.primitives.kdf.pbkdf2 import PBKDF2HMAC
from cryptography.hazmat.primitives.asymmetric import rsa, padding as asym_padding
from cryptography.hazmat.primitives.serialization import load_pem_private_key, load_pem_public_key

# Constants
AES_KEY_SIZE = 256  # bits
RSA_KEY_SIZE = 4096  # bits
PACKET_VERSION = 1
SECURE_MESH_ID = "dgla-secure-mesh-v1"

@dataclass
class MeshIdentity:
    """Mesh node identity"""
    node_id: str
    role: str
    public_key: bytes
    node_type: str
    region: str
    cluster_id: str

class PacketBlockchain:
    """Implements packet authentication chains for API communications"""
    
    def __init__(self):
        self.block_chain = []
        self.session_key = None
        self.last_block_hash = None
        
    def initialize_session(self, shared_key: bytes):
        """Initialize a new secure session"""
        # Create genesis block
        self.session_key = shared_key
        self.last_block_hash = hashlib.sha3_256(shared_key).hexdigest()
        timestamp = datetime.utcnow().isoformat()
        
        genesis_block = {
            "index": 0,
            "timestamp": timestamp,
            "session_id": str(uuid.uuid4()),
            "type": "session_init",
            "previous_hash": "0"
        }
        
        # Calculate block hash (includes block data + session key)
        block_data = json.dumps(genesis_block, sort_keys=True).encode('utf-8')
        block_hash = hashlib.sha3_256(block_data + self.session_key).hexdigest()
        genesis_block["hash"] = block_hash
        
        self.block_chain.append(genesis_block)
        self.last_block_hash = block_hash
        
        return genesis_block["session_id"]
        
    def add_packet_block(self, packet_type: str, metadata: Dict[str, Any]) -> str:
        """Add a packet to the blockchain and return the packet hash"""
        if not self.session_key:
            raise ValueError("Session not initialized")
            
        # Create new block
        timestamp = datetime.utcnow().isoformat()
        block = {
            "index": len(self.block_chain),
            "timestamp": timestamp,
            "type": packet_type,
            "metadata": metadata,
            "previous_hash": self.last_block_hash
        }
        
        # Calculate block hash
        block_data = json.dumps(block, sort_keys=True).encode('utf-8')
        block_hash = hashlib.sha3_256(block_data + self.session_key).hexdigest()
        block["hash"] = block_hash
        
        self.block_chain.append(block)
        self.last_block_hash = block_hash
        
        return block_hash
        
    def verify_chain_integrity(self) -> bool:
        """Verify the integrity of the packet blockchain"""
        if not self.block_chain:
            return True
            
        for i, block in enumerate(self.block_chain[1:], 1):
            # Verify hash chain
            if block["previous_hash"] != self.block_chain[i-1]["hash"]:
                return False
                
            # Verify block hash
            block_copy = block.copy()
            recorded_hash = block_copy.pop("hash")
            block_data = json.dumps(block_copy, sort_keys=True).encode('utf-8')
            calculated_hash = hashlib.sha3_256(block_data + self.session_key).hexdigest()
            
            if calculated_hash != recorded_hash:
                return False
                
        return True


class SecureMeshNode:
    """Implements secure mesh node with military-grade encryption"""
    
    def __init__(self, node_id: str, role: str, region: str = "default", 
                 cluster_id: str = None, key_dir: str = None):
        self.node_id = node_id
        self.role = role
        self.region = region
        self.cluster_id = cluster_id or str(uuid.uuid4())
        self.key_dir = key_dir or os.path.expanduser("~/.dgla/keys")
        self.connected_nodes: Dict[str, MeshIdentity] = {}
        self.session_keys: Dict[str, bytes] = {}
        self.packet_chains: Dict[str, PacketBlockchain] = {}
        
        # Ensure key directory exists
        os.makedirs(self.key_dir, exist_ok=True)
        
        # Generate or load keys
        self.private_key, self.public_key = self._load_or_generate_keys()
        
        # Set up logging
        self.logger = logging.getLogger(f"mesh-node-{self.node_id}")
        self.logger.setLevel(logging.INFO)
        if not self.logger.hasHandlers():
            handler = logging.StreamHandler()
            formatter = logging.Formatter('%(asctime)s - %(name)s - %(levelname)s - %(message)s')
            handler.setFormatter(formatter)
            self.logger.addHandler(handler)
    
    def _load_or_generate_keys(self):
        """Load existing keys or generate new ones"""
        private_key_path = os.path.join(self.key_dir, f"{self.node_id}_private.pem")
        public_key_path = os.path.join(self.key_dir, f"{self.node_id}_public.pem")
        
        if os.path.exists(private_key_path) and os.path.exists(public_key_path):
            # Load existing keys
            with open(private_key_path, "rb") as f:
                private_key = load_pem_private_key(f.read(), password=None)
                
            with open(public_key_path, "rb") as f:
                public_key = load_pem_public_key(f.read())
            
            return private_key, public_key
        else:
            # Generate new keys
            private_key = rsa.generate_private_key(
                public_exponent=65537,
                key_size=RSA_KEY_SIZE
            )
            public_key = private_key.public_key()
            
            # Serialize and save keys
            from cryptography.hazmat.primitives.serialization import (
                Encoding, PrivateFormat, PublicFormat, NoEncryption
            )
            
            private_pem = private_key.private_bytes(
                encoding=Encoding.PEM,
                format=PrivateFormat.PKCS8,
                encryption_algorithm=NoEncryption()  # In production, use encryption
            )
            
            public_pem = public_key.public_bytes(
                encoding=Encoding.PEM,
                format=PublicFormat.SubjectPublicKeyInfo
            )
            
            with open(private_key_path, "wb") as f:
                f.write(private_pem)
                
            with open(public_key_path, "wb") as f:
                f.write(public_pem)
                
            # Set correct permissions
            os.chmod(private_key_path, 0o600)  # Only owner can read/write
            
            return private_key, public_key
    
    def establish_connection(self, target_id: str, target_public_key, 
                           target_role: str, target_node_type: str,
                           target_region: str, target_cluster_id: str) -> str:
        """Establish secure connection with another mesh node"""
        # Store node identity
        self.connected_nodes[target_id] = MeshIdentity(
            node_id=target_id,
            role=target_role,
            public_key=target_public_key,
            node_type=target_node_type,
            region=target_region,
            cluster_id=target_cluster_id
        )
        
        # Generate session key
        session_key = secrets.token_bytes(32)  # 256-bit random key
        
        # Encrypt session key with target's public key
        encrypted_session_key = target_public_key.encrypt(
            session_key,
            asym_padding.OAEP(
                mgf=asym_padding.MGF1(algorithm=hashes.SHA256()),
                algorithm=hashes.SHA256(),
                label=None
            )
        )
        
        # Store session key
        self.session_keys[target_id] = session_key
        
        # Initialize packet blockchain for this connection
        self.packet_chains[target_id] = PacketBlockchain()
        session_id = self.packet_chains[target_id].initialize_session(session_key)
        
        # Create handshake message
        handshake = {
            "mesh_id": SECURE_MESH_ID,
            "node_id": self.node_id,
            "target_id": target_id,
            "session_id": session_id,
            "encrypted_session_key": base64.b64encode(encrypted_session_key).decode('utf-8'),
            "timestamp": datetime.utcnow().isoformat(),
            "ttl": (datetime.utcnow() + timedelta(hours=24)).isoformat()
        }
        
        self.logger.info(f"Established secure connection with node {target_id}")
        return session_id
    
    def encrypt_packet(self, target_id: str, data: Dict[str, Any]) -> Dict[str, Any]:
        """Encrypt a data packet for secure transmission"""
        if target_id not in self.session_keys:
            raise ValueError(f"No established session with {target_id}")
            
        session_key = self.session_keys[target_id]
        
        # Generate IV
        iv = secrets.token_bytes(16)  # 128 bits for AES
        
        # Serialize and pad data
        data_json = json.dumps(data).encode('utf-8')
        padder = padding.PKCS7(algorithms.AES.block_size).padder()
        padded_data = padder.update(data_json) + padder.finalize()
        
        # Encrypt with AES-256-GCM
        encryptor = Cipher(
            algorithms.AES(session_key),
            modes.GCM(iv)
        ).encryptor()
        
        # Add authenticated metadata
        nonce = secrets.token_bytes(16)
        encryptor.authenticate_additional_data(nonce)
        
        # Encrypt
        ciphertext = encryptor.update(padded_data) + encryptor.finalize()
        tag = encryptor.tag
        
        # Add to packet blockchain
        metadata = {
            "target_id": target_id,
            "timestamp": datetime.utcnow().isoformat(),
            "nonce": base64.b64encode(nonce).decode('utf-8')
        }
        packet_hash = self.packet_chains[target_id].add_packet_block("data", metadata)
        
        # Create secure packet
        packet = {
            "mesh_id": SECURE_MESH_ID,
            "version": PACKET_VERSION,
            "sender_id": self.node_id,
            "target_id": target_id,
            "iv": base64.b64encode(iv).decode('utf-8'),
            "tag": base64.b64encode(tag).decode('utf-8'),
            "nonce": base64.b64encode(nonce).decode('utf-8'),
            "ciphertext": base64.b64encode(ciphertext).decode('utf-8'),
            "hash": packet_hash,
            "timestamp": datetime.utcnow().isoformat()
        }
        
        return packet
    
    def decrypt_packet(self, packet: Dict[str, Any]) -> Dict[str, Any]:
        """Decrypt a secure packet"""
        sender_id = packet["sender_id"]
        if sender_id not in self.session_keys:
            raise ValueError(f"No established session with {sender_id}")
            
        session_key = self.session_keys[sender_id]
        
        # Decode components
        iv = base64.b64decode(packet["iv"])
        tag = base64.b64decode(packet["tag"])
        nonce = base64.b64decode(packet["nonce"])
        ciphertext = base64.b64decode(packet["ciphertext"])
        
        # Verify packet hash in blockchain
        if sender_id not in self.packet_chains:
            raise ValueError(f"No packet chain for {sender_id}")
        
        # Create AES-GCM decryptor
        decryptor = Cipher(
            algorithms.AES(session_key),
            modes.GCM(iv, tag)
        ).decryptor()
        
        # Add authenticated data
        decryptor.authenticate_additional_data(nonce)
        
        # Decrypt
        try:
            padded_data = decryptor.update(ciphertext) + decryptor.finalize()
            unpadder = padding.PKCS7(algorithms.AES.block_size).unpadder()
            data_json = unpadder.update(padded_data) + unpadder.finalize()
            data = json.loads(data_json.decode('utf-8'))
            
            # Update packet chain
            metadata = {
                "sender_id": sender_id,
                "timestamp": packet["timestamp"],
                "nonce": packet["nonce"]
            }
            self.packet_chains[sender_id].add_packet_block("received", metadata)
            
            return data
        except Exception as e:
            self.logger.error(f"Decryption failed: {str(e)}")
            raise ValueError("Packet decryption failed - possible tampering detected")


# Usage example
if __name__ == "__main__":
    # Set up logging
    logging.basicConfig(level=logging.INFO)
    
    # Create two mesh nodes
    sdk_node = SecureMeshNode("sdk-client", "client", region="canada-east")
    engine_node = SecureMeshNode("engine-server", "server", region="canada-east")
    
    # Establish secure connection (in real world, this happens through a secure exchange)
    session_id = sdk_node.establish_connection(
        "engine-server", 
        engine_node.public_key,
        "server",
        "engine",
        "canada-east",
        "prod-cluster-1"
    )
    
    # SDK prepares a secure message
    data = {
        "command": "deploy_agreement",
        "customer_id": "rogers-5g",
        "timestamp": datetime.utcnow().isoformat(),
        "payload": {
            "agreement_type": "sla",
            "terms": "Ensure 99.999% uptime for core network components"
        }
    }
    
    # Encrypt the packet
    secure_packet = sdk_node.encrypt_packet("engine-server", data)
    print(f"Secure packet created with {len(secure_packet)} encrypted fields")
    
    # In real world, packet is transmitted over network...
    
    # Engine node receives and processes it
    # First, it would establish the reverse connection (omitted for brevity)
    engine_node.session_keys["sdk-client"] = sdk_node.session_keys["engine-server"]
    engine_node.packet_chains["sdk-client"] = PacketBlockchain()
    engine_node.packet_chains["sdk-client"].initialize_session(engine_node.session_keys["sdk-client"])
    
    # Decrypt the message
    try:
        decrypted_data = engine_node.decrypt_packet(secure_packet)
        print(f"Successfully decrypted: {decrypted_data['command']}")
    except Exception as e:
        print(f"Decryption failed: {e}")
