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
    
    # Verify hooks are executable
    echo "📋 Verifying hook permissions..."
    ls -la .githooks/pre-commit .githooks/pre-push .githooks/commit-msg
    
    # Install hooks
    git config core.hooksPath .githooks
    
    # Verify hooks path is set
    echo "📋 Git hooks path: $(git config core.hooksPath)"
    
    # Test commit-msg hook with clearer logic
    echo "🧪 Testing commit-msg hook..."
    echo "Add new feature" > /tmp/test-commit-msg
    
    # Run the hook and capture the exit code
    if .githooks/commit-msg /tmp/test-commit-msg 2>/dev/null; then
        echo "❌ commit-msg hook test FAILED - invalid message was accepted!"
        echo "🔧 Hook may not be working properly"
    else
        echo "✅ commit-msg hook test PASSED - correctly rejected invalid message!"
    fi
    
    # Test with valid message
    echo "feat: add new feature" > /tmp/test-commit-msg-valid
    if .githooks/commit-msg /tmp/test-commit-msg-valid 2>/dev/null; then
        echo "✅ commit-msg hook accepts valid messages"
    else
        echo "❌ commit-msg hook incorrectly rejected valid message"
    fi
    
    rm -f /tmp/test-commit-msg /tmp/test-commit-msg-valid
    
    # Configure GPG
    echo "🔐 Configuring GPG..."
    
    # Ensure GPG directory exists and has correct permissions
    mkdir -p ~/.gnupg
    chmod 700 ~/.gnupg
    
    # Check if GPG keys are available
    if gpg --list-secret-keys --keyid-format=long | grep -q "sec"; then
        echo "🔐 GPG keys found. Configuring commit signing..."
        
        # Configure GPG agent for better passphrase caching
        echo "default-cache-ttl 28800" > ~/.gnupg/gpg-agent.conf   # 8 hours
        echo "max-cache-ttl 86400" >> ~/.gnupg/gpg-agent.conf      # 24 hours
        echo "pinentry-mode loopback" >> ~/.gnupg/gpg.conf
        
        # Configure Git for signing
        git config commit.gpgsign true
        git config tag.gpgsign true
        
        # Set GPG TTY
        echo 'export GPG_TTY=$(tty)' >> ~/.bashrc
        export GPG_TTY=$(tty)
        
        # Reload GPG agent
        gpgconf --reload gpg-agent
        
        echo "✅ Commit signing enabled with passphrase caching!"
    else
        echo "⚠️  No GPG keys found in devcontainer."
        echo "📝 To fix this:"
        echo "   1. Check that ~/.gnupg is properly mounted from host"
        echo "   2. Or import your GPG keys manually"
        echo "   3. Run this script again after fixing"
    fi
    
    echo "✅ Git hooks installed!"
fi

echo ""
echo "Hooks configured:"
echo "  📝 pre-commit: Format, lint, vet (no tests)"
echo "  🧪 pre-push: Full test suite with coverage"
echo "  📄 commit-msg: Enforce conventional commit format"
echo "  🔐 commit-signing: $([ -n "$(gpg --list-secret-keys)" ] && echo "Enabled" || echo "Disabled (no keys)")"