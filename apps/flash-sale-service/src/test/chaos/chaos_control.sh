#!/bin/bash

set -e

CONTAINER_NAME="${1:-redis}"
ACTION="${2:-status}"

echo "Chaos Engineering Control - $CONTAINER_NAME/$ACTION"

case "$CONTAINER_NAME" in
  redis)
    case "$ACTION" in
      kill)
        echo "Simulating Redis failure (kill container)..."
        docker kill redis-test-primary 2>/dev/null || echo "Container not running"
        ;;
      pause)
        echo "Pausing Redis container..."
        docker pause redis-test-primary 2>/dev/null || echo "Container not running"
        ;;
      unpause)
        echo "Resuming Redis container..."
        docker unpause redis-test-primary 2>/dev/null || echo "Container not running"
        ;;
      network-loss)
        echo "Simulating network loss for Redis..."
        docker network disconnect cuckoo_cuckoo-net redis-test-primary 2>/dev/null || echo "Failed"
        ;;
      network-restore)
        echo "Restoring network for Redis..."
        docker network connect cuckoo_cuckoo-net redis-test-primary 2>/dev/null || echo "Failed"
        ;;
      latency)
        LATENCY_MS="${3:-1000}"
        echo "Adding ${LATENCY_MS}ms latency to Redis..."
        # Using tc for network latency simulation
        docker exec redis-test-primary tc qdisc add dev eth0 root netem delay ${LATENCY_MS}ms 2>/dev/null || echo "Already configured"
        ;;
      status)
        docker ps --filter "name=redis" --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"
        ;;
      *)
        echo "Unknown action: $ACTION"
        exit 1
        ;;
    esac
    ;;
  kafka)
    case "$ACTION" in
    kill)
      echo "Simulating Kafka failure..."
      docker kill kafka-test 2>/dev/null || echo "Container not running"
      ;;
    pause)
      echo "Pausing Kafka..."
      docker pause kafka-test 2>/dev/null || echo "Container not running"
      ;;
    unpause)
      echo "Resuming Kafka..."
      docker unpause kafka-test 2>/dev/null || echo "Container not running"
      ;;
    status)
      docker ps --filter "name=kafka" --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"
      ;;
    *)
      echo "Unknown action: $ACTION"
      exit 1
      ;;
    esac
    ;;
  mysql)
    case "$ACTION" in
    kill)
      echo "Simulating MySQL failure..."
      docker kill mysql-test 2>/dev/null || echo "Container not running"
      ;;
    pause)
      echo "Pausing MySQL..."
      docker pause mysql-test 2>/dev/null || echo "Container not running"
      ;;
    unpause)
      echo "Resuming MySQL..."
      docker unpause mysql-test 2>/dev/null || echo "Container not running"
      ;;
    status)
      docker ps --filter "name=mysql" --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"
      ;;
    *)
      echo "Unknown action: $ACTION"
      exit 1
      ;;
    esac
    ;;
  *)
    echo "Unknown container: $CONTAINER_NAME"
    echo "Usage: $0 <redis|kafka|mysql> <kill|pause|unpause|status|network-loss|network-restore|latency> [ms]"
    exit 1
    ;;
esac

echo "Done"