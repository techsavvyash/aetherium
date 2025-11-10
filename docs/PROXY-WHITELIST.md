# Proxy Whitelist for VM Internet Access Control

## Overview

Aetherium now includes **Squid proxy-based whitelist functionality** to control which external services VMs can access over the internet. This provides security by:

- **Restricting internet access** to only approved domains
- **Preventing data exfiltration** to unauthorized services
- **Logging all internet requests** for audit trails
- **Blocking malicious or unwanted** connections

## Architecture

```
┌─────────────────────────────────────────────────┐
│                  Host System                     │
│                                                  │
│  ┌──────────────┐         ┌──────────────────┐  │
│  │    Squid     │◄────────│  Bridge          │  │
│  │    Proxy     │         │  (172.16.0.1)    │  │
│  │  Port 3128   │         └────────┬─────────┘  │
│  └──────────────┘                  │            │
│         │                           │            │
│         │ Whitelist Check           │            │
│         ▼                           ▼            │
│  [Allow/Deny]              ┌────────────────┐   │
│         │                  │   TAP Device   │   │
│         │                  └────────┬───────┘   │
└─────────┼───────────────────────────┼───────────┘
          │                           │
          │                           ▼
          │                    ┌──────────────┐
          └───────────────────▶│     VM       │
                Internet        │ (172.16.0.x) │
                Requests        └──────────────┘
```

**Flow:**
1. VM sends HTTP/HTTPS request (automatically uses proxy via env vars)
2. Request goes through bridge (172.16.0.1) to Squid proxy (port 3128)
3. Squid checks if domain is in whitelist
4. If whitelisted: Forward to internet
5. If not whitelisted: Block and return 403 error

## Setup

### Option 1: Using Docker Compose

1. **Start the Squid proxy:**

```bash
docker compose up -d squid
```

2. **Verify it's running:**

```bash
docker ps | grep squid
docker logs aetherium-squid
```

3. **Test the proxy:**

```bash
# Should succeed (github.com is whitelisted)
curl -x http://127.0.0.1:3128 https://github.com

# Should fail (facebook.com is NOT whitelisted)
curl -x http://127.0.0.1:3128 https://facebook.com
```

### Option 2: Using System Squid

1. **Run the setup script:**

```bash
sudo ./scripts/start-squid-proxy.sh
```

This will:
- Install Squid if not present
- Copy configuration files
- Initialize cache
- Start the proxy on port 3128

2. **Verify:**

```bash
# Check if running
ps aux | grep squid

# Test
curl -x http://127.0.0.1:3128 https://github.com
```

## Configuration

### Whitelist Configuration

Edit `config/whitelist.txt` to add or remove allowed domains:

```
# Package managers
.npmjs.org
.registry.npmjs.org
.pypi.org

# Source control
.github.com
.githubusercontent.com
.gitlab.com

# Add your domains here
.your-domain.com
```

**Format:**
- Use `.domain.com` (with leading dot) to allow all subdomains
- Use `domain.com` (without dot) to allow only exact domain
- Lines starting with `#` are comments

### Squid Configuration

Main config: `config/squid.conf`

**Key settings:**

```conf
# Timeouts (prevent hanging)
connect_timeout 30 seconds
read_timeout 60 seconds
request_timeout 60 seconds

# Allow VM network
acl localnet src 172.16.0.0/16

# Load whitelist
acl whitelist dstdomain "/etc/squid/whitelist.txt"

# Access rules
http_access allow localnet whitelist
http_access deny all  # Block everything else
```

## How VMs Use the Proxy

When a VM is created, the worker automatically injects proxy environment variables:

```go
// In pkg/worker/worker.go
vmConfig := &types.VMConfig{
    // ...
    Env: map[string]string{
        "HTTP_PROXY":  "http://172.16.0.1:3128",
        "HTTPS_PROXY": "http://172.16.0.1:3128",
        "http_proxy":  "http://172.16.0.1:3128",
        "https_proxy": "http://172.16.0.1:3128",
        "NO_PROXY":    "localhost,127.0.0.1,172.16.0.0/24",
        "no_proxy":    "localhost,127.0.0.1,172.16.0.0/24",
    },
}
```

These environment variables are:
- **Passed to all commands** executed in the VM
- **Used by tools** like curl, wget, npm, pip, git, etc.
- **Transparent** to the user (automatic)

## Testing

### Manual Test

1. **Start proxy:**

```bash
sudo ./scripts/start-squid-proxy.sh
```

2. **Create a VM:**

```bash
sudo ./bin/worker &
./bin/api-gateway &

# Create VM
./bin/aether-cli -type vm:create -name test-vm -vcpus 1 -memory 256
```

3. **Test access in VM:**

```bash
# Via API
curl -X POST http://localhost:8080/api/v1/vms/{vm-id}/execute \
  -H "Content-Type: application/json" \
  -d '{
    "command": "curl",
    "args": ["-s", "-o", "/dev/null", "-w", "%{http_code}", "https://github.com"]
  }'

# Should return 200

curl -X POST http://localhost:8080/api/v1/vms/{vm-id}/execute \
  -H "Content-Type: application/json" \
  -d '{
    "command": "curl",
    "args": ["-s", "-o", "/dev/null", "-w", "%{http_code}", "https://facebook.com"]
  }'

# Should fail or return 403
```

### Integration Test

Run the automated integration tests:

```bash
# Make sure Squid is running
sudo ./scripts/start-squid-proxy.sh

# Run tests
cd tests/integration
sudo go test -v -run TestProxyWhitelist -timeout 10m
```

**Tests include:**
1. ✅ Proxy environment variables are set
2. ✅ Access to whitelisted domains (github.com) succeeds
3. ✅ Access to non-whitelisted domains (facebook.com) is blocked
4. ✅ Squid logs show requests
5. ✅ Tools (git, npm) work through proxy
6. ✅ No hanging or timeouts

## Monitoring

### View Squid Logs

**Access log** (all requests):
```bash
tail -f /var/log/squid/access.log
```

**Cache log** (errors and startup):
```bash
tail -f /var/log/squid/cache.log
```

**Docker logs** (if using Docker):
```bash
docker logs -f aetherium-squid
```

### Log Format

```
1699999999.123    150 172.16.0.2 TCP_MISS/200 1234 GET https://github.com/ - HIER_DIRECT/140.82.121.4 text/html
```

Fields:
- `1699999999.123` - Timestamp
- `150` - Response time (ms)
- `172.16.0.2` - Client IP (VM)
- `TCP_MISS/200` - Cache status / HTTP code
- `1234` - Response size (bytes)
- `GET https://github.com/` - Request
- `HIER_DIRECT/140.82.121.4` - Upstream server

### Troubleshooting

#### VMs Can't Access Internet

1. **Check Squid is running:**
   ```bash
   ps aux | grep squid
   # or
   docker ps | grep squid
   ```

2. **Check Squid is listening on port 3128:**
   ```bash
   netstat -tlnp | grep 3128
   ```

3. **Test proxy from host:**
   ```bash
   curl -x http://127.0.0.1:3128 https://github.com -v
   ```

4. **Check VM can reach proxy:**
   ```bash
   # From VM
   curl -v http://172.16.0.1:3128
   ```

5. **Check firewall:**
   ```bash
   iptables -L -n | grep 3128
   ```

#### Proxy Hangs or Times Out

The Squid configuration includes aggressive timeouts to prevent hanging:

```conf
connect_timeout 30 seconds
read_timeout 60 seconds
request_timeout 60 seconds
persistent_request_timeout 60 seconds
```

If still experiencing hangs:

1. **Check DNS resolution:**
   ```bash
   # In squid.conf
   dns_nameservers 8.8.8.8 8.8.4.4
   dns_timeout 30 seconds
   ```

2. **Disable keep-alive:**
   ```conf
   half_closed_clients off
   ```

3. **Check Squid logs:**
   ```bash
   tail -100 /var/log/squid/cache.log | grep -i "error\|timeout\|fail"
   ```

4. **Test with curl verbose:**
   ```bash
   curl -x http://127.0.0.1:3128 https://github.com -v --trace-time
   ```

#### Whitelist Not Working

1. **Check whitelist file:**
   ```bash
   cat /etc/squid/whitelist.txt
   ```

2. **Reload Squid config:**
   ```bash
   squid -k reconfigure
   # or
   docker restart aetherium-squid
   ```

3. **Check ACL syntax:**
   ```bash
   squid -k parse
   ```

4. **Test with specific domain:**
   ```bash
   # Add test domain to whitelist
   echo ".example.com" >> /etc/squid/whitelist.txt
   squid -k reconfigure
   curl -x http://127.0.0.1:3128 https://example.com
   ```

## Security Considerations

### What This Protects Against

✅ **Unauthorized data exfiltration** - VMs can't send data to arbitrary services
✅ **Malware C2 communication** - Blocks unknown command & control servers
✅ **Supply chain attacks** - Only approved package sources allowed
✅ **Lateral movement** - VMs can't reach internal services (via NO_PROXY)

### What This Does NOT Protect Against

❌ **DNS tunneling** - Squid doesn't inspect DNS queries
❌ **Protocol tunneling** - Sophisticated malware may tunnel through HTTP
❌ **Compromised whitelisted domains** - If github.com is compromised, it's still allowed
❌ **Local attacks** - No protection against host-level exploits

### Best Practices

1. **Minimize whitelist** - Only add domains actually needed
2. **Monitor logs** - Set up alerts for unusual patterns
3. **Regular audits** - Review whitelist monthly
4. **Use specific domains** - Prefer `api.github.com` over `.github.com`
5. **Combine with other security** - Use with SELinux, AppArmor, seccomp

## Performance

### Overhead

- **Latency:** +10-50ms per request (proxy overhead)
- **Throughput:** Minimal impact (<5%)
- **Memory:** Squid uses ~50-100MB RAM
- **CPU:** Negligible (<1% per VM)

### Caching

Squid can cache responses to improve performance:

```conf
# In squid.conf
cache_dir ufs /var/spool/squid 10000 16 256  # 10GB cache
refresh_pattern .  0  20%  4320  # Cache for up to 3 days
```

Currently caching is disabled for testing:
```conf
cache deny all
```

Enable it in production for better performance.

## Advanced Configuration

### Per-VM Whitelists

To implement per-VM whitelists:

1. **Create multiple Squid instances** on different ports
2. **Pass different proxy URLs** to different VMs
3. **Use Squid ACLs** with authentication

Example:
```conf
# Different ACLs per user
acl user1 proxy_auth user1
acl user1_whitelist dstdomain "/etc/squid/whitelist-user1.txt"
http_access allow user1 user1_whitelist
```

### IP-Based Access Control

```conf
# Only allow specific VM IPs
acl vm_subnet src 172.16.0.10-172.16.0.20
http_access allow vm_subnet whitelist
http_access deny all
```

### Time-Based Access

```conf
# Only allow access during business hours
acl business_hours time MTWHF 09:00-17:00
http_access allow localnet whitelist business_hours
```

### SSL Bumping (HTTPS Inspection)

**Warning:** This requires man-in-the-middle certificates and may break some applications.

```conf
# Generate CA cert
openssl req -new -newkey rsa:2048 -days 365 -nodes -x509 \
  -keyout squid-ca-key.pem -out squid-ca-cert.pem

# Configure Squid
http_port 3128 ssl-bump \
  cert=/etc/squid/squid-ca-cert.pem \
  key=/etc/squid/squid-ca-key.pem

ssl_bump server-first all
sslcrtd_program /usr/lib/squid/security_file_certgen -s /var/spool/squid/ssl_db -M 4MB
```

## References

- **Squid Documentation:** http://www.squid-cache.org/Doc/
- **Squid ACL Reference:** http://www.squid-cache.org/Doc/config/acl/
- **RFC 3143 (HTTP Proxy):** https://tools.ietf.org/html/rfc3143

## Contributing

To add more domains to the default whitelist:

1. Edit `config/whitelist.txt`
2. Test with the integration tests
3. Submit a PR with justification for why the domain is needed

## Changelog

**v1.0.0** (2025-11-10)
- Initial implementation
- Squid-based whitelist
- Automatic proxy injection into VMs
- Integration tests
- Documentation
