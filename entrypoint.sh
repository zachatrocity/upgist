#!/bin/sh
eval $(ssh-agent -s)
if [ -d "/root/.ssh" ]; then
  for key in /root/.ssh/id_*; do
    if [ -f "$key" ] && [ "${key%.pub}" = "$key" ]; then
      ssh-add "$key"
    fi
  done
fi
exec "$@"
