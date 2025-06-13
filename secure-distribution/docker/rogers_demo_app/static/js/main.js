/**
 * DGLA Secure Infrastructure - Rogers Demo
 * Main JavaScript functionality
 */

document.addEventListener('DOMContentLoaded', function() {
    // Initialize tooltips
    var tooltipTriggerList = [].slice.call(document.querySelectorAll('[data-bs-toggle="tooltip"]'));
    tooltipTriggerList.map(function (tooltipTriggerEl) {
        return new bootstrap.Tooltip(tooltipTriggerEl);
    });

    // Dashboard card animations
    document.querySelectorAll('.dashboard-card').forEach(card => {
        card.addEventListener('mouseenter', () => {
            card.classList.add('shadow-lg');
        });
        
        card.addEventListener('mouseleave', () => {
            card.classList.remove('shadow-lg');
        });
    });

    // Security verification animation
    const verifyButton = document.getElementById('verify-integrity-btn');
    if (verifyButton) {
        verifyButton.addEventListener('click', function() {
            const statusBadge = document.getElementById('integrity-status');
            if (statusBadge) {
                statusBadge.textContent = 'Verifying...';
                statusBadge.classList.remove('bg-success');
                statusBadge.classList.add('bg-warning');
                
                setTimeout(() => {
                    statusBadge.textContent = 'Verified';
                    statusBadge.classList.remove('bg-warning');
                    statusBadge.classList.add('bg-success');
                }, 2000);
            }
        });
    }

    // Registry image search
    const searchInput = document.getElementById('search-images');
    if (searchInput) {
        searchInput.addEventListener('keyup', function() {
            const query = this.value.toLowerCase();
            
            document.querySelectorAll('.registry-image-row').forEach(row => {
                const imageName = row.querySelector('.image-name').textContent.toLowerCase();
                const imageTag = row.querySelector('.image-tag').textContent.toLowerCase();
                
                if (imageName.includes(query) || imageTag.includes(query)) {
                    row.style.display = '';
                } else {
                    row.style.display = 'none';
                }
            });
        });
    }

    // Deployment selection helpers
    document.querySelectorAll('.select-image-btn').forEach(button => {
        button.addEventListener('click', function() {
            const name = this.getAttribute('data-name');
            const tag = this.getAttribute('data-tag');
            
            document.getElementById('image_name').value = name;
            document.getElementById('image_tag').value = tag;
            
            // Scroll to the deployment form
            document.querySelector('.card-header:contains("Deploy NanoBond™-Verified Image")').scrollIntoView({
                behavior: 'smooth'
            });
        });
    });

    document.querySelectorAll('.select-cluster-btn').forEach(button => {
        button.addEventListener('click', function() {
            const id = this.getAttribute('data-id');
            document.getElementById('cluster_id').value = id;
            
            // Scroll to the deployment form
            document.querySelector('.card-header:contains("Deploy NanoBond™-Verified Image")').scrollIntoView({
                behavior: 'smooth'
            });
        });
    });

    // Auto refresh Kubernetes status
    const kubernetesStatusTable = document.querySelector('.kubernetes-status-table');
    if (kubernetesStatusTable) {
        // Refresh every 30 seconds
        setInterval(() => {
            fetch('/kubernetes_status')
                .then(response => response.text())
                .then(html => {
                    const parser = new DOMParser();
                    const doc = parser.parseFromString(html, 'text/html');
                    const newTable = doc.querySelector('.kubernetes-status-table');
                    
                    if (newTable) {
                        kubernetesStatusTable.innerHTML = newTable.innerHTML;
                    }
                })
                .catch(error => console.error('Error refreshing Kubernetes status:', error));
        }, 30000);
    }

    // Process dynamic data updates for simulations
    updateSimulationData();
});

// Update simulation data like timestamps and statuses
function updateSimulationData() {
    // Update timestamps to be relative
    document.querySelectorAll('.relative-time').forEach(element => {
        const timestamp = element.getAttribute('data-timestamp');
        if (timestamp) {
            const date = new Date(timestamp);
            const now = new Date();
            const diffSeconds = Math.floor((now - date) / 1000);
            
            if (diffSeconds < 60) {
                element.textContent = `${diffSeconds} seconds ago`;
            } else if (diffSeconds < 3600) {
                const minutes = Math.floor(diffSeconds / 60);
                element.textContent = `${minutes} ${minutes === 1 ? 'minute' : 'minutes'} ago`;
            } else if (diffSeconds < 86400) {
                const hours = Math.floor(diffSeconds / 3600);
                element.textContent = `${hours} ${hours === 1 ? 'hour' : 'hours'} ago`;
            } else {
                const days = Math.floor(diffSeconds / 86400);
                element.textContent = `${days} ${days === 1 ? 'day' : 'days'} ago`;
            }
        }
    });
}

// Generate random hexadecimal hash for simulations
function generateRandomHash(length = 64) {
    const characters = '0123456789abcdef';
    let result = '';
    for (let i = 0; i < length; i++) {
        result += characters.charAt(Math.floor(Math.random() * characters.length));
    }
    return result;
}

// Simulate cryptographic verification
function simulateVerification(elementId, successProbability = 0.99) {
    const element = document.getElementById(elementId);
    if (!element) return;
    
    element.innerHTML = '<div class="spinner-border spinner-border-sm" role="status"><span class="visually-hidden">Loading...</span></div> Verifying...';
    
    setTimeout(() => {
        const success = Math.random() < successProbability;
        
        if (success) {
            element.innerHTML = '<span class="text-success">✓ Verified</span>';
        } else {
            element.innerHTML = '<span class="text-danger">✗ Failed</span>';
        }
    }, 1500);
}
