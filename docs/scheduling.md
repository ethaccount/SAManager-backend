# SAManager Scheduling Service Architecture

## Overview

The SAManager Scheduling Service is a core component of the SAManager-backend that provides automated blockchain execution capabilities. It consists of two main services: the Polling Service that monitors smart account execution configurations stored on-chain, and the Execution Service that executes scheduled UserOperations using a single private key to sign all transactions.

This scheduling service is one part of the larger SAManager-backend system, which may also include other services such as passkey relying party servers and additional API endpoints. This document focuses specifically on the scheduling functionality.

## Core Concepts

- **Single Private Key**: One private key signs all UserOperations for all users
- **Blockchain as Source of Truth**: Execution configs are stored on-chain, backend only stores job mappings
- **Polling-based Execution**: Continuously polls blockchain storage to detect overdue executions
- **ERC-4337 Integration**: Submits signed UserOperations to external bundler services

## System Architecture

### Component Diagram

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   SAManager     │    │   Blockchain    │    │   ERC-4337      │
│     Dapp        │    │   Contract      │    │   Bundler       │
└─────────┬───────┘    └─────────┬───────┘    └─────────┬───────┘
          │                      │                      │
          │ Register Jobs        │ Read Storage         │ Submit UserOps
          │                      │                      │
          ▼                      ▼                      ▲
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   API Service   │    │ Polling Service │    │Execution Service│
└─────────┬───────┘    └─────────┬───────┘    └─────────┬───────┘
          │                      │                      │
          └──────────────────────┼──────────────────────┘
                                 │
                                 ▼
                    ┌─────────────────┐
                    │   PostgreSQL    │
                    │   Database      │
                    └─────────────────┘
```

### Service Responsibilities

#### API Service
- **Purpose**: Job registration and management interface
- **Functions**:
  - Register job mappings (account_address, job_id, user_operation)
  - Store single private key in encrypted keystore
  - Provide job status queries
  - Handle authentication

#### Polling Service
- **Purpose**: Monitor blockchain and trigger executions
- **Functions**:
  - Read `executionLog` mapping from blockchain contract
  - Calculate execution timing based on on-chain data
  - Detect overdue executions
  - Trigger execution service when jobs are due

#### Execution Service
- **Purpose**: Sign and submit UserOperations
- **Functions**:
  - Retrieve UserOperation from database
  - Sign with the single private key
  - Submit to ERC-4337 bundler
  - Handle retry logic for failed submissions

## Data Models

### Blockchain Storage

```solidity
// On-chain execution configuration storage
mapping(address accountAddress => mapping(uint256 jobId => ExecutionConfig)) public executionLog;

struct ExecutionConfig {
    uint48 executeInterval;                 // Seconds between executions
    uint16 numberOfExecutions;              // Total planned executions
    uint16 numberOfExecutionsCompleted;     // Completed execution count
    uint48 startDate;                      // Job start timestamp
    bool isEnabled;                        // Job active status
    uint48 lastExecutionTime;              // Last execution timestamp
    bytes executionData;                   // Additional execution parameters
}
```

### Database Schema

#### Job Mappings
```sql
CREATE TABLE registered_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_address VARCHAR(42) NOT NULL,
    job_id BIGINT NOT NULL,
    user_operation JSONB NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(account_address, job_id)
);

CREATE INDEX idx_smart_account_job ON registered_jobs(account_address, job_id);
```

#### Single Private Key Storage
```sql
CREATE TABLE keystore (
    id INTEGER PRIMARY KEY DEFAULT 1,
    encrypted_private_key TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    CONSTRAINT single_key_only CHECK (id = 1)
);
```

### UserOperation Format (ERC-4337)

```json
{
    "jsonrpc": "2.0",
    "method": "eth_sendUserOperation",
    "params": [
        {
            "sender": "0x5a6b47F4131bf1feAFA56A05573314BcF44C9149",
            "nonce": "0x845ADB2C711129D4F3966735ED98A9F09FC4CE5700000000000000000000",
            "factory": "0xd703aaE79538628d27099B8c4f621bE4CCd142d5",
            "factoryData": "0xc5265d5d000000000000000000000000aac5d4240af87249b3f71bc8e4a2cae074a3e419...",
            "callData": "0xe9ae5c5300000000000000000000000000000000000000000000000000000000000000000000000000...",
            "callGasLimit": "0x13880",
            "verificationGasLimit": "0x60B01",
            "preVerificationGas": "0xD3E3",
            "maxPriorityFeePerGas": "0x3B9ACA00",
            "maxFeePerGas": "0x7A5CF70D5",
            "paymaster": "0x",
            "paymasterVerificationGasLimit": "0x0",
            "paymasterPostOpGasLimit": "0x0",
            "paymasterData": null,
            "signature": "0xa6cc6589c8bd561cfd68d7b6b0757ef6f208e7438782939938498eee7d703260137856c840c491b3d415956265e81bf5c2184a725be2abfc365f7536b6af525e1c"
        },
        "0x0000000071727De22E5E9d8BAf0edAc6f37da032"
    ],
    "id": 1
}
```

## System Flows

### 1. Job Registration
```
SAManager Dapp → API Service: POST /jobs/register
├── Parameters: account_address, job_id, user_operation
├── API Service → Database: Store job mapping
└── Response: success/failure

Note: Execution config is stored on blockchain by the dapp
```

### 2. Execution Detection & Triggering
```
Polling Service (continuous loop):
├── Read executionLog[account_address][job_id] from blockchain
├── For each registered job:
│   ├── Check: isEnabled && (lastExecutionTime + executeInterval < now)
│   └── If overdue → Trigger Execution Service
└── Sleep interval, repeat
```

### 3. UserOperation Execution
```
Execution Service receives trigger:
├── Retrieve user_operation from database
├── Retrieve private key from keystore
├── Sign user_operation with private key
├── Submit signed UserOp to ERC-4337 bundler
└── Handle response/retry if needed
```

## Configuration

```yaml
# Environment configuration
database:
  host: ${DB_HOST}
  port: ${DB_PORT}
  name: ${DB_NAME}
  max_connections: 10

blockchain:
  rpc_url: ${BLOCKCHAIN_RPC_URL}
  contract_address: ${CONTRACT_ADDRESS}
  polling_interval: 30s

bundler:
  url: ${BUNDLER_URL}
  timeout: 30s

execution:
  max_retries: 3
  retry_delay: 5s

security:
  jwt_secret: ${JWT_SECRET}
  encryption_key: ${ENCRYPTION_KEY}
```

## Security

### Private Key Management
- **Single Key**: One private key for all UserOperation signing
- **Encryption**: AES-256-GCM encryption at rest
- **Access**: Only execution service can access the key
- **Storage**: Encrypted in database with application-level decryption

### API Security
- **Authentication**: JWT tokens for API access
- **TLS**: All external communications encrypted
- **Validation**: Input validation on all endpoints

## Error Handling & Retry

### Retry Strategy
```go
type RetryConfig struct {
    MaxAttempts int           // Default: 3
    RetryDelay  time.Duration // Default: 5s
}
```

### Error Categories
- **Network Errors**: Retry with exponential backoff
- **Invalid UserOp**: Log error, skip execution
- **Gas Issues**: Retry with adjusted gas parameters
- **Bundler Errors**: Retry with different bundler endpoint

## Monitoring

### Logging
```json
{
    "timestamp": "2024-01-15T10:30:00Z",
    "level": "info",
    "service": "execution",
    "account_address": "0x...",
    "job_id": 42,
    "action": "userop_submitted",
    "user_op_hash": "0x..."
}
```

### Health Checks
- `/health/database` - Database connectivity
- `/health/blockchain` - RPC endpoint status
- `/health/bundler` - Bundler service availability

## Project Structure

```
SAManager-backend/
├── cmd/
│   ├── api/           # API service main
│   ├── polling/       # Polling service main
│   └── execution/     # Execution service main
├── src/
│   ├── domain/        # Core business entities
│   ├── repository/    # Database access layer
│   ├── service/       # Business logic
│   ├── handler/       # HTTP request handlers
│   ├── blockchain/    # Blockchain client
│   ├── keystore/      # Private key management
│   └── bundler/       # ERC-4337 bundler client
├── migrations/        # Database schema migrations
├── config/           # Configuration files
└── README.md
```

## Development & Testing

### Testing Strategy
- **Unit Tests**: Business logic and utilities
- **Integration Tests**: Database and blockchain interactions
- **E2E Tests**: Full workflow from registration to execution

### Local Development
1. Start PostgreSQL database
2. Run migrations
3. Set environment variables
4. Start services: `make run-api`, `make run-polling`, `make run-execution`

## Risk Considerations

### Critical Risks
- **Private Key Compromise**: Single point of failure for all transactions
- **Blockchain RPC Failure**: Service becomes non-functional
- **Database Corruption**: Loss of job mappings

### Mitigation Strategies
- **Key Security**: Hardware security module for production
- **RPC Redundancy**: Multiple blockchain RPC endpoints
- **Database Backup**: Regular automated backups
- **Monitoring**: Real-time alerting for service failures

This architecture provides a solid MVP foundation while maintaining simplicity and the ability to scale as requirements evolve. 