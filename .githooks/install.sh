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
    
    # Set GPG TTY for container/host
    export GPG_TTY=$(tty)
    echo "export GPG_TTY=\$(tty)" >> ~/.bashrc
    
    # Check if we're in a container environment
    if [ -f "/.dockerenv" ] || [ -n "$REMOTE_CONTAINERS" ] || [ -n "$CODESPACES" ]; then
        echo "📦 Running in container environment"
        
        # Use vscode user's home directory
        GPG_HOME="$HOME/.gnupg"
        
        # Simple GPG configuration for container
        mkdir -p "$GPG_HOME"
        chmod 700 "$GPG_HOME"
        
        cat > "$GPG_HOME/gpg.conf" << EOF
use-agent
pinentry-mode loopback
no-tty
batch
EOF
        
        cat > "$GPG_HOME/gpg-agent.conf" << EOF
default-cache-ttl 28800
max-cache-ttl 86400
allow-loopback-pinentry
pinentry-program /usr/bin/pinentry-curses
EOF
        
    else
        # Host environment configuration
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
    fi
    
    # Restart GPG agent
    echo "🔄 Restarting GPG agent..."
    gpgconf --kill gpg-agent 2>/dev/null || true
    sleep 1
    gpg-agent --daemon 2>/dev/null || true
    
    # Check if GPG keys are available
    if gpg --list-secret-keys --keyid-format=long | grep -q "sec"; then
        echo "🔐 GPG keys found. Enabling commit signing..."
        
        # Configure Git for signing
        git config commit.gpgsign true
        git config tag.gpgsign true
        
        # Test GPG signing
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