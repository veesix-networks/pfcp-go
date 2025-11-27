<<<<<<< HEAD
# PFCP-GO

PFCP (Packet Forwarding Control Protocol) implementation in Go based on 3GPP TS 29.244 (not fully compliant/more influenced rather than a strict implementation that follows the spec).

## Configuration

### Control Plane (pfcp-cp)

```bash
pfcp-cp [flags]
```

**Flags:**
- `-node-id` - PFCP Control Plane node ID (default: `cp-node-1`)
- `-listen-addr` - Listen address for PFCP protocol (default: `:8805`)
- `-grpc-addr` - gRPC API address for northbound interface (default: `:50051`)
- `-heartbeat-interval` - Heartbeat interval (default: `60s`)
- `-retransmit-n1` - Max retransmission attempts (default: `3`)
- `-retransmit-t1` - Retransmission timeout (default: `3s`)

**Example:**
```bash
pfcp-cp \
  -node-id=cp-node-1 \
  -listen-addr=0.0.0.0:8805 \
  -grpc-addr=0.0.0.0:50051 \
  -heartbeat-interval=60s
```

### User Plane (pfcp-up)

```bash
pfcp-up [flags]
```

**Flags:**
- `-node-id` - PFCP User Plane node ID (default: `up-node-1`)
- `-cp-address` - Control Plane address (default: `127.0.0.1:8805`)
- `-local-addr` - Local listen address for PFCP protocol (default: `:8805`)
- `-heartbeat-interval` - Heartbeat interval (default: `60s`)
- `-dataplane` - Dataplane type: `vpp` or `mock` (default: `vpp`)
- `-vpp-socket` - VPP API socket path (default: `/run/vpp/api.sock`)

**Example (VPP dataplane):**
```bash
pfcp-up \
  -node-id=up-node-1 \
  -cp-address=pfcp-cp:8805 \
  -local-addr=0.0.0.0:8805 \
  -dataplane=vpp \
  -vpp-socket=/run/vpp/api.sock
```

**Example (Mock dataplane):**
```bash
pfcp-up \
  -node-id=up-node-1 \
  -cp-address=127.0.0.1:8805 \
  -dataplane=mock
```

## Docker Deployment

The project includes Docker Compose configuration for testing with VPP dataplane.

### Quick Start

```bash
docker compose -f test/docker/docker-compose.yml up --build
```

This starts:
- **pfcp-cp** - Control Plane on port 50052 (gRPC) and 8805 (PFCP)
- **pfcp-up-vpp** - User Plane with VPP dataplane

### Port Mappings

- `50052/tcp` - gRPC API for northbound control (mapped from container's 50051)
- `8805/udp` - PFCP protocol

### VPP Requirements

The VPP user plane container requires:
- Privileged mode for VPP dataplane access
- Host volumes: `/sys/bus/pci/drivers`, `/sys/kernel/mm/hugepages`, `/sys/devices/system/node`, `/dev`
- Capabilities: `SYS_ADMIN`, `SYS_NICE`, `SYS_RESOURCE`, `NET_ADMIN`, `IPC_LOCK`

## Punting

You can use [grpcurl](https://github.com/fullstorydev/grpcurl) to create PFCP rules on the control plane.

## DHCP packets (L3/L4 with SDF Filter)

```bash
grpcurl -plaintext -proto api/pfcp/v1/control.proto -d '{
  "node_id": "up-node-1",
  "pdrs": [{
    "id": 1,
    "precedence": 1000,
    "pdi": {
      "source_interface": 0,
      "sdf_filter": "permit in udp from any to any 67-68"
    },
    "far_id": 1
  }],
  "fars": [{
    "id": 1,
    "apply_action": 2,
    "forwarding_params": {"destination_interface": 2}
  }]
}' localhost:50052 pfcp.v1.ControlPlane/CreateSession
```

Output:
```
pfcp-up-vpp  | VPP: Installing PDR 1 for session 1 (precedence: 0, FAR_ID: 1)
pfcp-up-vpp  | VPP: Installing FAR 1 for session 1 (action: 0x02)
pfcp-up-vpp  | VPP: Configuring L4 punt for PDR 1 (flow: permit in udp from any to any 67-68, protocol=17, ports=67-68, af=0)
pfcp-up-vpp  | VPP: L4 punt registered for protocol=17 port=67
pfcp-up-vpp  | VPP: L4 punt registered for protocol=17 port=68
pfcp-up-vpp  | VPP: Punt configured for PDR 1
pfcp-cp      | gRPC: Session created SEID=1 for node up-node-1
```

## ARP (L2 with Application ID)

```bash
grpcurl -plaintext -proto api/pfcp/v1/control.proto -d '{
  "node_id": "up-node-1",
  "pdrs": [{
    "id": 2,
    "precedence": 2000,
    "pdi": {
      "source_interface": 0,
      "application_id": "ARP"
    },
    "far_id": 2
  }],
  "fars": [{
    "id": 2,
    "apply_action": 2,
    "forwarding_params": {"destination_interface": 2}
  }]
}' localhost:50052 pfcp.v1.ControlPlane/CreateSession
```

Output:
```
pfcp-up-vpp  | VPP: Installing PDR 2 for session 2 (precedence: 0, FAR_ID: 2)
pfcp-up-vpp  | VPP: Installing FAR 2 for session 2 (action: 0x02)
pfcp-up-vpp  | VPP: Configuring L2 punt for PDR 2 (Application ID: ARP, EtherType: 0x0806)
pfcp-up-vpp  | VPP: L2 punt configured for Application ID ARP (EtherType 0x0806)
pfcp-cp      | gRPC: Session created SEID=2 for node up-node-1
```

## PPPoE Discovery (L2 with Application ID)

```bash
grpcurl -plaintext -proto api/pfcp/v1/control.proto -d '{
  "node_id": "up-node-1",
  "pdrs": [{
    "id": 3,
    "precedence": 2000,
    "pdi": {
      "source_interface": 0,
      "application_id": "PPPOE_DISCOVERY"
    },
    "far_id": 3
  }],
  "fars": [{
    "id": 3,
    "apply_action": 2,
    "forwarding_params": {"destination_interface": 2}
  }]
}' localhost:50052 pfcp.v1.ControlPlane/CreateSession
```

## Available Application IDs

Pre-configured L2 filters (from `pkg/dataplane/vpp/l2_filters.go`):
- `ARP` - EtherType 0x0806
- `PPPOE_DISCOVERY` - EtherType 0x8863
- `PPPOE_SESSION` - EtherType 0x8864
- `LLDP` - EtherType 0x88cc
- `DOT1Q` - EtherType 0x8100
- `IPV6` - EtherType 0x86dd

## Architecture

### PFCP Protocol (3GPP TS 29.244)

PFCP enables Control and User Plane Separation (CUPS) for packet gateways. The Control Plane manages session state and policy, while the User Plane handles high-speed packet forwarding.

**Control Plane (CP):**
- Manages PFCP associations with User Plane nodes
- Creates/modifies/deletes packet forwarding sessions
- Sends heartbeats to monitor User Plane health
- Provides gRPC northbound API for configuration

**User Plane (UP):**
- Establishes association with Control Plane
- Installs packet detection and forwarding rules
- Processes packets according to PDRs (Packet Detection Rules)
- Applies FARs (Forwarding Action Rules) - forward, drop, or notify CP

### Session Rules

A PFCP session consists of:

**PDR (Packet Detection Rule):**
- Matches packets using PDI (Packet Detection Information)
- PDI contains: source interface, SDF filter, or Application ID
- References FAR to apply when packets match

**FAR (Forwarding Action Rule):**
- Defines action: DROP (0x01), FORW (forward), or NOCP (notify CP) (0x02)
- Contains forwarding parameters (destination interface, network instance)

**SDF Filter vs Application ID:**
- **SDF Filter** - L3/L4 matching using flow descriptions (IP 5-tuple: src/dst IP, src/dst port, protocol)
- **Application ID** - L2 matching using pre-configured filters (EtherType-based for ARP, PPPoE, etc.)

**Note on L2 Protocol Handling:**

PFCP (3GPP TS 29.244) does not natively define L2 protocol matching. Typically, user plane implementations handle ARP, Neighbor Discovery (IPv6 ND), and other L2 protocols through separate out-of-band punt paths that are configured independently from PFCP sessions.

This implementation extends PFCP using the Application ID IE (section 8.2.6) to provide a unified abstraction for both L3/L4 and L2 punt rules. This allows the control plane to manage all punt rules—including ARP, PPPoE Discovery, and other L2 protocols—through the standard PFCP session establishment flow, rather than requiring separate configuration mechanisms. While not strictly compliant with the base PFCP specification, this approach aligns with the spec's support for application-specific filters and simplifies deployment for BNG/CUPS use cases (TR-459).

### VPP Dataplane

VPP (Vector Packet Processing) integration uses:
- **govpp** - Go bindings for VPP binary API
- **L4 punt** - For TCP/UDP/SCTP with port ranges (via `SetPunt` with `PUNT_API_TYPE_L4`)
- **IP proto punt** - For other IP protocols like GRE, ESP, L2TP (via `SetPunt` with `PUNT_API_TYPE_IP_PROTO`)
- **L2 punt (TODO)** - For L2 protocols via classify tables with EtherType matching

## Project Structure

```
pfcp-go/
├── api/pfcp/v1/          # gRPC protobuf definitions
├── cmd/
│   ├── pfcp-cp/          # Control Plane main
│   └── pfcp-up/          # User Plane main
├── pkg/
│   ├── cp/               # Control Plane logic
│   ├── up/               # User Plane logic
│   ├── protocol/         # PFCP protocol encoding/decoding
│   └── dataplane/        # Dataplane implementations
│       ├── mock/         # Mock dataplane for testing
│       └── vpp/          # VPP dataplane integration
├── test/docker/          # Docker Compose test environment
└── docs/                 # 3GPP specifications and notes
```
=======
# pfcp-go
Packet Filter Configuration Protocol - Go implementation for client/server side
>>>>>>> 98f14f9d488f772bba68913ab90cc9d2c64974fc
