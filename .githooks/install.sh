#!/bin/bash
# Install git hooks

echo "Installing Git hooks..."

# Check if we're in a git repository
if ! git rev-parse --git-dir > /dev/null 2>&1; then
    echo "⚠️  Not in a git repository. Git hooks will be configured when repo is initialized."
else
    # Make hooks executable
    chmod +x .githooks/pre-commit
    chmod +x .githooks/pre-push
    chmod +x .githooks/commit-msg
    
    # Install hooks (project-specific setting)
    git config core.hooksPath .githooks
    
    # Configure GPG
    echo "Configuring GPG..."
    
    # Set GPG TTY for container/host
    export GPG_TTY=$(tty)
    echo "export GPG_TTY=\$(tty)" >> ~/.bashrc
    
    # Check if we're in a container environment
    if [ -f "/.dockerenv" ] || [ -n "$REMOTE_CONTAINERS" ] || [ -n "$CODESPACES" ]; then
        echo "Running in container environment"
        
        # Keys are already in the image, just set up GPG agent
        GPG_HOME="$HOME/.gnupg"
        
        # Configure GPG for container use
        cat > "$GPG_HOME/gpg.conf" << 'EOF'
use-agent
pinentry-mode loopback
no-tty
batch
trust-model always
disable-dirmngr
keyserver-options no-honor-keyserver-url
keyserver-options no-auto-key-retrieve
EOF
        
        cat > "$GPG_HOME/gpg-agent.conf" << 'EOF'
default-cache-ttl 28800
max-cache-ttl 86400
allow-loopback-pinentry
pinentry-program /usr/bin/pinentry-curses
disable-scdaemon
no-grab
EOF
        
        chmod 600 "$GPG_HOME/gpg.conf" "$GPG_HOME/gpg-agent.conf" 2>/dev/null || true
        
        # Kill existing GPG processes
        echo "Restarting GPG agent..."
        pkill -f gpg-agent 2>/dev/null || true
        gpgconf --kill all 2>/dev/null || true
        sleep 2
        
        # Start GPG agent
        gpg-agent --daemon --allow-loopback-pinentry 2>/dev/null &
        sleep 2
        
    else
        # Host environment configuration
        mkdir -p ~/.gnupg
        chmod 700 ~/.gnupg
        chmod 600 ~/.gnupg/* 2>/dev/null || true
        
        cat > ~/.gnupg/gpg.conf << 'EOF'
use-agent
pinentry-mode loopback
EOF
        
        cat > ~/.gnupg/gpg-agent.conf << 'EOF'
default-cache-ttl 28800
max-cache-ttl 86400
allow-loopback-pinentry
EOF
    fi
    
    # Check if GPG keys are available
    if gpg --list-secret-keys --keyid-format=long | grep -q "sec"; then
        echo "GPG keys found. Enabling commit signing..."
        
        # Get the key ID
        KEY_ID=$(gpg --list-secret-keys --keyid-format=long | grep "sec" | head -1 | sed 's/.*\/\([A-F0-9]*\).*/\1/')
        echo "Using key ID: $KEY_ID"
        
        # Configure Git for signing (PROJECT-LEVEL only if different from global)
        GLOBAL_KEY=$(git config --global user.signingkey 2>/dev/null || echo "")
        if [ "$KEY_ID" != "$GLOBAL_KEY" ]; then
            echo "Setting project-specific signing key: $KEY_ID"
            git config user.signingkey "$KEY_ID"
        fi
        
        # Only set local config if different from global
        GLOBAL_SIGN=$(git config --global commit.gpgsign 2>/dev/null || echo "false")
        if [ "$GLOBAL_SIGN" != "true" ]; then
            git config commit.gpgsign true
            git config tag.gpgsign true
        fi
        
        # Test GPG signing
        echo "Testing GPG signing..."
        
        if echo "test" | gpg --clearsign --armor --pinentry-mode loopback --batch --passphrase "" --local-user "$KEY_ID" >/dev/null 2>&1; then
            echo "✅ GPG signing test successful!"
        else
            echo "⚠️  GPG key appears to have a passphrase. Consider using a key without passphrase for automated signing."
            
            # Disable automatic signing for this project only
            git config commit.gpgsign false
            git config tag.gpgsign false
            echo "GPG configured but automatic signing disabled due to passphrase"
        fi
        
    else
        echo "⚠️  No GPG keys found."
        # Only set local config if global doesn't already disable it
        GLOBAL_SIGN=$(git config --global commit.gpgsign 2>/dev/null || echo "false")
        if [ "$GLOBAL_SIGN" = "true" ]; then
            git config commit.gpgsign false
            git config tag.gpgsign false
        fi
    fi
    
    echo "✅ Git hooks installed!"
fi

echo ""
echo "Hooks configured:"
echo "  pre-commit: Format, lint, vet (no tests)"
echo "  pre-push: Full test suite with coverage"
echo "  commit-msg: Enforce conventional commit format"
echo "  commit-signing: $(git config commit.gpgsign || echo 'false')"