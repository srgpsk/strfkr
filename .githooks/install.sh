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
    
    # Install hooks
    git config core.hooksPath .githooks
    
    # Configure GPG
    echo "🔐 Configuring GPG..."
    
    # Check if we're in a container environment
    if [ -f "/.dockerenv" ] || [ -n "$REMOTE_CONTAINERS" ] || [ -n "$CODESPACES" ]; then
        echo "📦 Running in container environment"
        
        # Use container-specific GPG home to avoid affecting host
        export GNUPGHOME="/tmp/.gnupg-container"
        mkdir -p "$GNUPGHOME"
        chmod 700 "$GNUPGHOME"
        
        # Set GPG TTY for container
        export GPG_TTY=$(tty)
        echo "export GPG_TTY=\$(tty)" >> ~/.bashrc
        echo "export GNUPGHOME=\"$GNUPGHOME\"" >> ~/.bashrc
        
        # Configure GPG for container use with loopback pinentry
        cat > "$GNUPGHOME/gpg.conf" << EOF
use-agent
pinentry-mode loopback
no-tty
batch
EOF
        
        # Configure GPG agent for container
        cat > "$GNUPGHOME/gpg-agent.conf" << EOF
default-cache-ttl 28800
max-cache-ttl 86400
allow-loopback-pinentry
pinentry-program /usr/bin/pinentry-curses
EOF
        
        # Kill and restart GPG agent with proper error handling
        echo "🔄 Restarting GPG agent..."
        gpgconf --kill gpg-agent 2>/dev/null || true
        sleep 1
        gpg-agent --daemon 2>/dev/null || true
        
        # Import keys from host if available (but don't modify host directory)
        if [ -d "/root/.gnupg" ] && [ -f "/root/.gnupg/secring.gpg" -o -f "/root/.gnupg/private-keys-v1.d" ]; then
            echo "🔑 Importing GPG keys from host..."
            # Copy keys without changing host permissions
            if [ -f "/root/.gnupg/secring.gpg" ]; then
                cp "/root/.gnupg/secring.gpg" "$GNUPGHOME/" 2>/dev/null || true
            fi
            if [ -f "/root/.gnupg/pubring.gpg" ]; then
                cp "/root/.gnupg/pubring.gpg" "$GNUPGHOME/" 2>/dev/null || true
            fi
            if [ -d "/root/.gnupg/private-keys-v1.d" ]; then
                cp -r "/root/.gnupg/private-keys-v1.d" "$GNUPGHOME/" 2>/dev/null || true
            fi
            if [ -f "/root/.gnupg/trustdb.gpg" ]; then
                cp "/root/.gnupg/trustdb.gpg" "$GNUPGHOME/" 2>/dev/null || true
            fi
        fi
        
    else
        # Host environment configuration - use default GPG home
        # Ensure GPG directory exists and has correct permissions
        mkdir -p ~/.gnupg
        chmod 700 ~/.gnupg
        chmod 600 ~/.gnupg/* 2>/dev/null || true
        
        cat > ~/.gnupg/gpg.conf << EOF
use-agent
pinentry-mode loopback
EOF
        
        cat > ~/.gnupg/gpg-agent.conf << EOF
default-cache-ttl 28800
max-cache-ttl 86400
allow-loopback-pinentry
EOF
        
        # Set GPG TTY environment variable
        echo 'export GPG_TTY=$(tty)' >> ~/.bashrc
        export GPG_TTY=$(tty)
        
        # Restart GPG agent
        gpgconf --kill gpg-agent
        gpg-agent --daemon
    fi
    
    # Check if GPG keys are available
    if gpg --list-secret-keys --keyid-format=long | grep -q "sec"; then
        echo "🔐 GPG keys found. Enabling commit signing..."
        
        # Configure Git for signing
        git config commit.gpgsign true
        git config tag.gpgsign true
        
        # Test GPG signing with loopback mode
        echo "🧪 Testing GPG signing..."
        if echo "test" | gpg --clearsign --armor --pinentry-mode loopback >/dev/null 2>&1; then
            echo "✅ GPG signing test successful!"
        else
            echo "⚠️  GPG signing test failed. Disabling commit signing..."
            git config commit.gpgsign false
            git config tag.gpgsign false
        fi
        
    else
        echo "⚠️  No GPG keys found."
        git config commit.gpgsign false
        git config tag.gpgsign false
    fi
    
    echo "✅ Git hooks installed!"
fi

echo ""
echo "Hooks configured:"
echo "  📝 pre-commit: Format, lint, vet (no tests)"
echo "  🧪 pre-push: Full test suite with coverage"
echo "  📄 commit-msg: Enforce conventional commit format"
echo "  🔐 commit-signing: $(git config commit.gpgsign || echo 'false')"