#!/bin/bash
set -e

echo "Starting VPP test container..."

PFCP_CP_ADDRESS="${PFCP_CP_ADDRESS:-172.20.0.10:8805}"
PFCP_UP_NODE_ID="${PFCP_UP_NODE_ID:-up-node-1}"
PFCP_UP_LOCAL_ADDR="${PFCP_UP_LOCAL_ADDR:-:8805}"

setup_hugepages() {
    echo "Setting up hugepages..."
    mkdir -p /dev/hugepages
    if ! mount | grep -q hugetlbfs; then
        mount -t hugetlbfs -o pagesize=2M none /dev/hugepages || true
    fi
    echo 256 > /proc/sys/vm/nr_hugepages || true
}

setup_interfaces() {
    echo "Setting up test interfaces..."

    ip link add vpp-access type veth peer name access-host || true
    ip link set vpp-access up
    ip link set access-host up
    ip addr add 10.10.10.1/24 dev access-host || true
}

start_vpp() {
    echo "Starting VPP..."
    /usr/bin/vpp -c ${VPP_STARTUP_CONF} &

    sleep 5

    if ! pgrep vpp > /dev/null; then
        echo "ERROR: VPP failed to start"
        exit 1
    fi

    echo "VPP started successfully"
}

configure_vpp() {
    echo "Configuring VPP interfaces..."

    vppctl create host-interface name vpp-access || true
    vppctl set interface state host-vpp-access up || true
    vppctl set interface ip address host-vpp-access 10.10.10.254/24 || true

    vppctl show interface || true
    vppctl show interface addr || true
}

start_pfcp_up() {
    echo "Starting PFCP UP function..."
    echo "  Node ID: ${PFCP_UP_NODE_ID}"
    echo "  CP Address: ${PFCP_CP_ADDRESS}"
    echo "  Local Address: ${PFCP_UP_LOCAL_ADDR}"

    exec /usr/local/bin/pfcp-up \
        -node-id "${PFCP_UP_NODE_ID}" \
        -cp-address "${PFCP_CP_ADDRESS}" \
        -local-addr "${PFCP_UP_LOCAL_ADDR}"
}

setup_hugepages
setup_interfaces
start_vpp
configure_vpp
start_pfcp_up
