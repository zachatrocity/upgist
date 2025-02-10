#!/bin/sh

# Start SSH agent
eval $(ssh-agent -s)

if [ -d "/root/.ssh" ]; then
  # Create askpass script with provided passphrase or empty if not set
  echo '#!/bin/sh' > /usr/local/bin/ssh-askpass
  if [ ! -z "$SSH_PASSPHRASE" ]; then
    echo "echo \"$SSH_PASSPHRASE\"" >> /usr/local/bin/ssh-askpass
  else
    echo "echo ''" >> /usr/local/bin/ssh-askpass
  fi
  chmod +x /usr/local/bin/ssh-askpass
  export SSH_ASKPASS=/usr/local/bin/ssh-askpass
  export DISPLAY=:0

  # Add all private keys
  for key in /root/.ssh/id_*; do
    if [ -f "$key" ] && [ "${key%.pub}" = "$key" ]; then
      ssh-add "$key" < /dev/null
    fi
  done
fi

exec "$@"
