#!/bin/bash

# FizHub Deployment Script

# Check if rsync is installed
if ! command -v rsync &> /dev/null; then
    echo "Error: rsync is required but not installed."
    exit 1
fi

# Configuration
REMOTE_USER="fiz"
REMOTE_HOST="fiznode.local"
REMOTE_DIR="/home/fiz/fizhub"
LOCAL_DIR="."

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${GREEN}Starting FizHub deployment to $REMOTE_USER@$REMOTE_HOST${NC}"

# Create remote directory if it doesn't exist
ssh $REMOTE_USER@$REMOTE_HOST "mkdir -p $REMOTE_DIR"

# Sync files to Raspberry Pi
echo "Syncing files..."
rsync -avz --exclude '.git' \
    --exclude 'deploy.sh' \
    --exclude '.DS_Store' \
    $LOCAL_DIR/ $REMOTE_USER@$REMOTE_HOST:$REMOTE_DIR/

# Check if rsync was successful
if [ $? -eq 0 ]; then
    echo -e "${GREEN}Files synced successfully${NC}"
else
    echo -e "${RED}Error syncing files${NC}"
    exit 1
fi

# Install dependencies and build on Raspberry Pi
echo "Setting up FizHub on Raspberry Pi..."
ssh $REMOTE_USER@$REMOTE_HOST "cd $REMOTE_DIR && \
    # Install Go if not already installed
    if ! command -v go &> /dev/null; then
        echo 'Installing Go...'
        sudo apt-get update
        sudo apt-get install -y golang
    fi && \
    # Build the application
    go build -o fizhub cmd/fizhub/main.go && \
    # Create systemd service
    sudo tee /etc/systemd/system/fizhub.service > /dev/null <<EOL
[Unit]
Description=FizHub Service
After=network.target

[Service]
Type=simple
User=fiz
WorkingDirectory=$REMOTE_DIR
ExecStart=$REMOTE_DIR/fizhub
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOL
    # Reload systemd and start service
    sudo systemctl daemon-reload && \
    sudo systemctl enable fizhub && \
    sudo systemctl restart fizhub"

# Check deployment status
if [ $? -eq 0 ]; then
    echo -e "${GREEN}FizHub has been successfully deployed!${NC}"
    echo -e "\nYou can check the service status with:"
    echo "ssh $REMOTE_USER@$REMOTE_HOST 'sudo systemctl status fizhub'"
    echo -e "\nView logs with:"
    echo "ssh $REMOTE_USER@$REMOTE_HOST 'sudo journalctl -u fizhub -f'"
else
    echo -e "${RED}Error during deployment${NC}"
    exit 1
fi
