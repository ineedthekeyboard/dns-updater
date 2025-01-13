# DNS Updater

This project is a simple DNS updater that updates a DNS record with your current IP address using the DigitalOcean API.

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

3. **Edit the `.env` file** with your DigitalOcean API token, domain, and DNS record ID:

    ```env
    DO_API_TOKEN=your_digitalocean_api_token
    DO_DOMAIN=your_domain.com
    DO_RECORD_ID=your_dns_record_id
    ```

## Building the Project

To build the project for Windows, you can use the provided build script:

```sh
./windowBuildScript.sh
```