---
name: hivarr-router
description: Manage and optimize the Hivarr 5G router running OpenWrt. Use this skill when the user asks about their router, internet connection, network configuration, QoS, bufferbloat, speed issues, or wants to run commands on the router.
allowed-tools: Bash
---

# Hivarr 5G Router Management

## Router Details

- **Brand**: Hivarr (rebranded Chinese 5G CPE, web UI branded "WTAG")
- **Firmware**: OpenWrt 21.02-SNAPSHOT (2.5.0) on MediaTek MT7981 (Filogic 820)
- **CPU**: ARM Cortex-A53 (aarch64)
- **RAM**: ~232 MB
- **Gateway IP**: 192.168.88.1
- **Access**: Unauthenticated root telnet on port 23 (no SSH)
- **Web UI**: http://192.168.88.1 (Angular app, limited functionality)

## Network Topology

- **Internet**: Vodafone 5G cellular via `wwan0_1` interface
- **LAN**: `br-lan` (192.168.88.0/24) bridging `lan1-lan4` + WiFi
- **Guest**: `br-guest` (192.168.89.0/24)
- **WAN config name**: `lte0` (maps to physical `wwan0_1`)
- **DNS**: 8.8.8.8, 8.8.4.4 (Google)
- **WiFi radios**: `ra0`/`ra1` (2.4GHz), `rax0`/`rax1` (5GHz WiFi 6)

## Connecting to the Router

Use `expect` with `nc` for reliable scripted commands:

```bash
expect -c '
set timeout 30
spawn nc 192.168.88.1 23
expect "OpenWrt"
expect "#"
send "YOUR_COMMAND_HERE\r"
expect "#"
send "exit\r"
expect eof
'
```

For multiple commands, chain `send`/`expect` pairs:

```bash
expect -c '
set timeout 30
spawn nc 192.168.88.1 23
expect "OpenWrt"
expect "#"
send "command1\r"
expect "#"
send "command2\r"
expect "#"
send "exit\r"
expect eof
'
```

Important: Do NOT use single quotes inside the expect script (UCI values don't need them). Do NOT call `/etc/init.d/sqos status` as it has a bug that stops the service.

## QoS / Traffic Shaping (sqos)

The router has a vendor QoS system called `sqos` using HFSC + fq_codel.

### Current Configuration

- **Enabled**: Yes, on `lte0` interface
- **Download limit**: 500000 kbps (effectively unlimited — ingress policing doesn't work well on 5G)
- **Upload limit**: 18000 kbps (shapes egress to prevent bufferbloat)
- **Qdisc**: fq_codel leaves under HFSC hierarchy
- **ECN**: Enabled for both ingress and egress

### Key QoS Commands

```
# View config
uci show sqos

# Change upload shaping (kbps)
uci set sqos.lte0.upload=18000
uci commit sqos

# Change download limit (kbps)
uci set sqos.lte0.download=500000
uci commit sqos

# Restart QoS (must stop then start, not restart)
/etc/init.d/sqos stop 2>&1
# wait 2 seconds
/etc/init.d/sqos start 2>&1

# Verify QoS is active
tc qdisc show dev wwan0_1
tc class show dev wwan0_1

# Disable QoS
uci set sqos.lte0.enabled=0
uci commit sqos
/etc/init.d/sqos stop
```

### Traffic Classes (HFSC)

| Class | Name | Bandwidth Share | Description |
|-------|------|----------------|-------------|
| 1 | RealTime | 15% (min 200kbps up, 500kbps down) | Small packets, VoIP, ACKs |
| 2 | Fast | 60% | HTTP/HTTPS (ports 80, 443, 8080) |
| 3 | Slow | 5% | Bulk transfers (large packets 1300-1500 bytes) |
| 4 | Normal | 20% (default) | Everything else |

### 5G-Specific Notes

- **Ingress (download) shaping is ineffective on 5G** because buffering happens in the carrier's radio network, not on the router. Set the download limit very high (500Mbps+) to avoid artificially capping throughput.
- **Egress (upload) shaping works well** because the router controls when packets leave. Set to ~95-100% of measured upload speed.
- **5G speeds are highly variable** — they fluctuate with time of day, cell congestion, weather, and signal conditions. Don't be alarmed by speed variations between tests.
- The `xt_set` iptables module is missing (causes harmless error messages about ipset on sqos start — can be ignored).

## Common Diagnostic Commands

```
# Network interfaces and IPs
ip addr show wwan0_1
ip route show

# Check active traffic shaping
tc qdisc show dev wwan0_1
tc class show dev wwan0_1
tc -s class show dev wwan0_1   # with statistics

# Firewall rules
iptables -t mangle -L -n

# System info
free -m
cat /proc/cpuinfo
cat /etc/openwrt_release

# Interface traffic counters
cat /proc/net/dev

# DNS config
uci show network.lte0
cat /etc/config/network

# WiFi status
iwconfig ra0 2>/dev/null
iwconfig rax0 2>/dev/null

# Connected clients
cat /tmp/dhcp.leases

# View all UCI config
uci show

# Check what packages are installed
opkg list-installed
```

## Speed Testing

Run from the local machine (not the router):

```bash
# Quick test
speedtest --accept-license --accept-gdpr

# Test against a specific server for consistent comparison
speedtest --accept-license --accept-gdpr --server-id 67540
```

Server 67540 is FibreNest London, good for consistent UK benchmarks.

## TCP and Kernel Tuning (persisted in /etc/rc.local)

The following optimizations are applied at boot via `/etc/rc.local`:

```
# TCP buffer tuning for high-speed 5G (default was 208KB max)
sysctl -w net.core.rmem_max=4194304
sysctl -w net.core.wmem_max=4194304
sysctl -w net.ipv4.tcp_rmem="4096 262144 4194304"
sysctl -w net.ipv4.tcp_wmem="4096 65536 4194304"

# TCP MTU probing (cellular PMTU discovery can be unreliable)
sysctl -w net.ipv4.tcp_mtu_probing=1

# TCP Fast Open for client and server
sysctl -w net.ipv4.tcp_fastopen=3

# Distribute ethernet IRQs across both CPU cores (default: all on CPU 0)
echo 2 > /proc/irq/76/smp_affinity
echo 2 > /proc/irq/78/smp_affinity
echo 2 > /proc/irq/81/smp_affinity
```

### What each tuning does

| Setting | Default | Tuned | Purpose |
|---------|---------|-------|---------|
| rmem_max / wmem_max | 208 KB | 4 MB | Max socket buffer size — allows filling the bandwidth-delay product on high-speed links |
| tcp_rmem | 4K/128K/1.8M | 4K/256K/4M | TCP receive autotuning range (min/default/max) |
| tcp_wmem | 4K/16K/1.8M | 4K/64K/4M | TCP send autotuning range |
| tcp_mtu_probing | 0 (off) | 1 (on) | Probes for working MTU, avoids black-hole routing on cellular |
| tcp_fastopen | 1 (client) | 3 (client+server) | Saves a round-trip on repeat TCP connections |
| IRQ affinity | All CPU 0 | Split CPU 0+1 | Distributes network interrupt processing across both cores |

### Not available / not applicable

- **TCP BBR**: Not compiled into kernel (only `reno` and `cubic` available). Cannot install via opkg.
- **CAKE / cake-autorate**: Not installed. Cannot install via opkg. Would be the ideal solution for variable 5G bandwidth.
- **Flow offloading**: Kernel modules present (`kmod-ipt-offload`, `kmod-nf-flow`) but MUST NOT be enabled — offloaded flows bypass `tc` qdiscs, which breaks QoS.
- **DNS change**: Google DNS (8.8.8.8) benchmarked fastest from this location (~34ms vs Cloudflare ~37ms).

## Baseline Performance (Vodafone 5G, measured 2026-02-17)

Best results when 5G signal is good (varies significantly with time of day and conditions):

| Metric | Without QoS | With QoS + Tuning |
|--------|------------|----------|
| Idle Latency | 44 ms (jitter 33ms) | 32-35 ms (jitter 3-8ms) |
| Download | 119 Mbps | 66-136 Mbps (5G variable) |
| Upload | 18.5 Mbps | 10-13 Mbps |
| Upload Latency | 753 ms | 100-135 ms |
| Upload Worst | 1,756 ms | 260-320 ms |
| Packet Loss | 1.4% | 0.0% |

## Package Management

The opkg package feeds are **broken** (custom firmware, repos return 404). You cannot install new packages via `opkg install`. Any optimization must use what's already on the router. Key missing packages that would help if a firmware update makes them available: `kmod-sched-cake`, `sqm-scripts`, `kmod-tcp-bbr`.

## Security Notes

- Telnet is unauthenticated root access — anyone on the LAN can connect
- The web UI at 192.168.88.1 has no HTTPS
- Consider disabling telnet after configuration is stable if security is a concern
