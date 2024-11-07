# FizHub

FizHub is the central hub component of the Fiz system, designed to run on a Raspberry Pi. It manages communication between Fiz Readers and the Cursive server.

## System Requirements

- Raspberry Pi (3 or newer recommended)
- Raspbian/Raspberry Pi OS
- Go 1.16 or newer
- Network connectivity

## Quick Start

Run the application locally:
```bash
go run .
```

## Installation on Raspberry Pi

1. Make the deployment script executable:
```bash
chmod +x deploy.sh
```

2. Run the deployment script:
```bash
./deploy.sh
```

The script will:
- Copy all necessary files to the Raspberry Pi
- Install required dependencies
- Build the application
- Set up and start the FizHub service

## Service Management

Check service status:
```bash
ssh fiz@fiznode.local 'sudo systemctl status fizhub'
```

View logs:
```bash
ssh fiz@fiznode.local 'sudo journalctl -u fizhub -f'
```

Restart service:
```bash
ssh fiz@fiznode.local 'sudo systemctl restart fizhub'
```

## Configuration

The configuration file is located at `configs/config.json`. Key settings:

```json
{
  "server": {
    "port": "8080"
  },
  "cursive": {
    "url": "http://nfc.cursive.team",
    "timeout": "30s"
  }
}
```

## Features

- NFC tag UID collection and buffering
- Communication with Fiz Readers over Wi-Fi
- Integration with Cursive API
- State management and visual feedback
- Power management for connected devices

## Architecture

FizHub operates on a hybrid star network topology:

1. **Fiz Readers**: Connect to FizHub over Wi-Fi
2. **State Management**: Handles UID collection and validation
3. **Cursive Integration**: Manages API communication
4. **Power Management**: Optimizes device power consumption

## UID Processing

1. Collects UIDs from Fiz Readers
2. Formats UIDs according to Cursive specification:
   ```
   https://nfc.cursive.team/tap?uid=<UID>
   ```
3. Buffers up to three UIDs before validation
4. Sends validation request to Cursive API:
   ```json
   {
     "uids": [
       "ec586341127a6414",
       "another_uid",
       "yet_another_uid"
     ]
   }
   ```

## Development

### Running Locally

1. Start the application:
```bash
go run .
```

2. Test endpoints:
```bash
# Check status
curl http://localhost:8080/api/status

# Send UID
curl -X POST http://localhost:8080/api/receive_uid \
  -H "Content-Type: application/json" \
  -d '{"uid": "ec586341127a6414"}'
```

## Troubleshooting

1. If the service fails to start:
   ```bash
   ssh fiz@fiznode.local 'sudo journalctl -u fizhub -n 50'
   ```

2. Check network connectivity:
   ```bash
   ssh fiz@fiznode.local 'ping -c 4 nfc.cursive.team'
   ```

3. Verify configuration:
   ```bash
   ssh fiz@fiznode.local 'cat /home/fiz/fizhub/configs/config.json'
