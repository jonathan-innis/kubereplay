# kubereplay

A kubectl CLI plugin that extracts relevant state information for Kubernetes objects from audit logs. It can parse audit logs from local files or AWS CloudWatch Logs to identify key events such as pod creation, binding, Karpenter nominations, and status changes.

## Usage

### Basic syntax
```bash
kubereplay get <resource> <name> -n <namespace> [flags]
```

### Supported resources
- `pod` - Get events for a specific pod

### Data sources
- `--audit-log` or `-f` - Local audit log file path
- `--log-group` or `-g` - AWS CloudWatch log group name

### Examples

```bash
# Get pod events from local audit log file
kubereplay get pod my-pod -n kube-system -f /var/log/audit.log

# Get pod events from AWS CloudWatch Logs
kubereplay get pod my-pod -n default -g /aws/eks/my-cluster/audit

# Get pod events from default namespace
kubereplay get pod my-pod -n default -f /path/to/audit.log
```

## Installation

### Prerequisites
- Go 1.21 or later
- kubectl installed and configured

### Build from source

```bash
# Clone the repository
git clone https://github.com/joinnis/kubereplay.git
cd kubereplay

# Build the binary
make build

# Install to /usr/local/bin (requires sudo)
make install
```

### Manual installation

```bash
# Build and copy manually
make build
cp bin/kubereplay /usr/local/bin/
```

## Development

```bash
# Run tests
make test

# Clean build artifacts
make clean

# Build
make build
```
