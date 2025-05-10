package merkle

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"
)

// MerkleNode represents a node in the Merkle tree
type MerkleNode struct {
	Hash  string
	Left  *MerkleNode
	Right *MerkleNode
	Data  interface{}
}

// MerkleTree represents a Merkle tree structure for creating cryptographic proofs
type MerkleTree struct {
	Root      *MerkleNode
	DataNodes []*MerkleNode
}

// MerkleProof represents a cryptographic proof for data in the Merkle tree
type MerkleProof struct {
	Root        string    `json:"merkle_root" yaml:"merkle_root"`
	GeneratedAt time.Time `json:"generated_at" yaml:"generated_at"`
	DataHashes  []string  `json:"data_hashes,omitempty" yaml:"data_hashes,omitempty"`
}

// NewMerkleTree creates a new Merkle tree from the provided data items
func NewMerkleTree(dataItems []interface{}) (*MerkleTree, error) {
	if len(dataItems) == 0 {
		return nil, fmt.Errorf("cannot create a Merkle tree with no data")
	}
	
	// Create leaf nodes
	var nodes []*MerkleNode
	for _, item := range dataItems {
		node := newLeafNode(item)
		nodes = append(nodes, node)
	}
	
	dataNodes := make([]*MerkleNode, len(nodes))
	copy(dataNodes, nodes)
	
	// Build the tree by creating parent nodes
	for len(nodes) > 1 {
		var levelUp []*MerkleNode
		
		for i := 0; i < len(nodes); i += 2 {
			var right *MerkleNode
			left := nodes[i]
			
			// If we have an odd number of nodes at this level, duplicate the last one
			if i+1 < len(nodes) {
				right = nodes[i+1]
			} else {
				right = left
			}
			
			// Create a parent node
			parentHash := hashChildren(left.Hash, right.Hash)
			parent := &MerkleNode{
				Hash:  parentHash,
				Left:  left,
				Right: right,
			}
			
			levelUp = append(levelUp, parent)
		}
		
		nodes = levelUp
	}
	
	return &MerkleTree{
		Root:      nodes[0],
		DataNodes: dataNodes,
	}, nil
}

// GenerateProof creates a Merkle proof for the tree
func (m *MerkleTree) GenerateProof() MerkleProof {
	dataHashes := make([]string, len(m.DataNodes))
	for i, node := range m.DataNodes {
		dataHashes[i] = node.Hash
	}
	
	return MerkleProof{
		Root:        m.Root.Hash,
		GeneratedAt: time.Now(),
		DataHashes:  dataHashes,
	}
}

// GetRoot returns the Merkle root hash
func (m *MerkleTree) GetRoot() string {
	if m.Root == nil {
		return ""
	}
	return m.Root.Hash
}

// VerifyData checks if the data is included in the Merkle tree
func (m *MerkleTree) VerifyData(data interface{}) bool {
	hash := calculateHash(data)
	
	for _, node := range m.DataNodes {
		if node.Hash == hash {
			return true
		}
	}
	
	return false
}

// newLeafNode creates a new leaf node for the Merkle tree
func newLeafNode(data interface{}) *MerkleNode {
	hash := calculateHash(data)
	return &MerkleNode{
		Hash: hash,
		Data: data,
	}
}

// calculateHash creates a SHA-256 hash for the given data
func calculateHash(data interface{}) string {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return ""
	}
	
	hash := sha256.Sum256(jsonData)
	return hex.EncodeToString(hash[:])
}

// hashChildren combines and hashes the hashes of two child nodes
func hashChildren(left, right string) string {
	combined := left + right
	hash := sha256.Sum256([]byte(combined))
	return hex.EncodeToString(hash[:])
}
