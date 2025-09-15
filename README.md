# kubereplay

A kubectl CLI plugin that extracts relevant state information for Kubernetes objects from audit logs. It can parse audit logs from local files or AWS CloudWatch Logs to identify key events such as pod creation, binding, Karpenter nominations, and status changes.

## Usage

### Commands
- `get` - Get Kubernetes resources from audit log events
- `describe` - Describe audit log events for Kubernetes resources

### Basic syntax
```bash
kubereplay get <resource> <name> [flags]
kubereplay describe <resource> <name> [flags]
```

### Supported resources
- `pod` - Get/describe events for a specific pod
- `node` - Get/describe events for a specific node

### Data sources
- `--audit-log` or `-f` - Local audit log file path
- `--log-group` or `-g` - AWS CloudWatch log group name
- `--region` or `-r` - AWS region for CloudWatch log group
- `--start` - Start time for log parsing (duration format, default: 24h)
- `--end` - End time for log parsing (duration format, default: 0)

### Examples

```bash
# Get pod YAML from local audit log file
kubereplay get pod my-pod -n kube-system -f /var/log/audit.log

# Get pod YAML from AWS CloudWatch Logs
kubereplay get pod my-pod -n default -g /aws/eks/my-cluster/audit -r us-west-2

# Get node YAML from audit logs
kubereplay get node i-0871709ffb35ae35b -g /aws/eks/cluster-name/audit

# Describe pod events from audit logs
kubereplay describe pod my-pod -n default -f /path/to/audit.log

# Describe node events from audit logs
kubereplay describe node i-0871709ffb35ae35b -g /aws/eks/cluster-name/audit
```

## Installation

### Prerequisites
- Go 1.24 or later
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
