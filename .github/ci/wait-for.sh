#!/bin/sh
set -e

host="$1"
port="$2"
shift 2

while ! nc -z "$host" "$port"; do
  echo "Waiting for $host:$port..."
  sleep 1
done

exec "$@"