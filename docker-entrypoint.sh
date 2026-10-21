#!/bin/sh
set -e

# If the first argument is a known command or executable, execute it
if [ "$1" = "tinystoredb-cli" ] || [ "$1" = "tinystoredb" ] || [ "$1" = "tinystoredb-server" ] || [ "$1" = "sh" ] || [ "$1" = "bash" ]; then
    exec "$@"
fi

# Otherwise, default to starting the server with any passed flags
exec tinystoredb-server "$@"
