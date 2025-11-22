# DNS Updater

This project is a simple DNS updater that updates DNS records with your current IP address using the DigitalOcean API. It supports updating multiple subdomains simultaneously.

## Setup

1. **Clone the repository:**

    ```sh
    git clone https://github.com/ineedthekeyboard/dns-updater.git
    cd dns-updater
    ```

2. **Copy the example environment file:**

    ```sh
    cp example.env .env
    ```

3. **Edit the `.env` file** with your DigitalOcean API token and domains:

    ```env
    DO_API_TOKEN=your_digitalocean_api_token
    DO_DOMAINS=suba.domain.net,subb.domain.net,subc.domain.net
    UPDATE_MINUTES=5
    ```

    You can specify:
    - **Single domain**: `DO_DOMAINS=sub.domain.net`
    - **Multiple domains** (comma-separated): `DO_DOMAINS=sub1.domain.net,sub2.domain.net,sub3.domain.net`

    All domains will be updated with the same IP address.

## Building the Project

### Build All Platforms

To build binaries for all platforms (Windows, macOS, Linux, Raspberry Pi):

```sh
./allVersionBuilsScript.sh
```

This creates binaries in the `dist/` directory for:
- **Windows** (64-bit & 32-bit)
- **macOS** (Intel & Apple Silicon)
- **Linux** (64-bit)
- **Linux ARM64** (Raspberry Pi 3/4/5 with 64-bit OS)

### Build Windows Only

To build only for Windows:

```sh
./windowBuildScript.sh
```

## Running the Application

### Run Locally

After building, run the appropriate binary for your platform:

```sh
# Automatically detect and run the correct binary
./allVersionBuilsScript.sh --run

# Or run a specific binary
./dist/do-ddns_linux_amd64        # Linux 64-bit
./dist/do-ddns_darwin_arm64       # macOS Apple Silicon
./dist/do-ddns_windows_amd64.exe  # Windows 64-bit
```

### Run on Raspberry Pi

1. Copy the ARM64 binary to your Raspberry Pi:
   ```sh
   scp dist/do-ddns_linux_arm64 pi@your-pi-address:~/
   ```

2. On your Raspberry Pi, make it executable and run:
   ```sh
   chmod +x do-ddns_linux_arm64
   ./do-ddns_linux_arm64
   ```

3. (Optional) Set up as a systemd service to run on boot - see the systemd section below.

## Running as a Service (Linux/Raspberry Pi)

To run the updater as a background service that starts automatically:

1. Create a systemd service file:
   ```sh
   sudo nano /etc/systemd/system/ddns-updater.service
   ```

2. Add the following content (adjust paths as needed):
   ```ini
   [Unit]
   Description=DDNS Updater Service
   After=network.target

   [Service]
   Type=simple
   User=pi
   WorkingDirectory=/home/pi
   ExecStart=/home/pi/do-ddns_linux_arm64
   Restart=always
   RestartSec=10

   [Install]
   WantedBy=multi-user.target
   ```

3. Enable and start the service:
   ```sh
   sudo systemctl daemon-reload
   sudo systemctl enable ddns-updater.service
   sudo systemctl start ddns-updater.service
   ```

4. Check the status:
   ```sh
   sudo systemctl status ddns-updater.service
   ```

5. View logs:
   ```sh
   sudo journalctl -u ddns-updater.service -f
   ```
