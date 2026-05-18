#!/bin/sh
set -eu

if [ "$#" -eq 0 ]; then
    set -- /usr/local/bin/wg -c /config/config.yaml
elif [ "${1#-}" != "$1" ]; then
    set -- /usr/local/bin/wg "$@"
fi

if [ "${1:-}" = "/usr/local/bin/wg" ]; then
    need_config=1
    config_path=/config/config.yaml
    prev=

    for arg in "$@"; do
        case "$arg" in
            -g|-pg|-h)
                need_config=0
                ;;
        esac

        if [ "$prev" = "-c" ]; then
            config_path="$arg"
        fi
        prev="$arg"
    done

    if [ "$need_config" -eq 1 ] && [ ! -c /dev/net/tun ]; then
        echo "WireGold requires /dev/net/tun inside the container." >&2
        echo "Run with: --device /dev/net/tun --cap-add NET_ADMIN" >&2
        echo "If Network is icmp or ip, add --cap-add NET_RAW as well." >&2
        exit 1
    fi

    if [ "$need_config" -eq 1 ] && [ ! -f "$config_path" ]; then
        echo "WireGold config not found: $config_path" >&2
        echo "Mount your config into /config or override -c." >&2
        exit 1
    fi
fi

exec "$@"
