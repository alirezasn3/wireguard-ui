# Wireguard UI

Wireguard UI is a web-based interface designed to simplify the management of Wireguard VPN servers. It provides a user-friendly dashboard for administrators to manage VPN peers and groups, monitor usage, and handle configurations.

## Features

- **Peer Management**: Easily add, remove, and edit VPN peers.
- **Group Management**: Organize peers into groups for simplified management.
- **Configuration Management**: Generate and download configuration files for peers directly from the UI.
- **Usage Statistics**: Monitor bandwidth usage and other statistics for the VPN server and its peers.
- **Real-time Updates**: The dashboard updates peer information in real-time, without the need for page refreshes.
- **QR Code Generation**: Generate QR codes for peer configurations, facilitating easy setup of Wireguard clients.

## User Roles

In Wireguard UI, there are two main types of users: Admins and Normal Peers. Here's how they differ:

### Admin Peers

- **Full Access**: Admins have full access to the Wireguard UI dashboard and can manage all aspects of the VPN server.
- **Peer Management**: They can add, remove, and edit any peer within the VPN network.
- **Group Management**: Admins can create and manage groups, assigning peers to these groups as needed.
- **Server Configuration**: They have the ability to change server settings and manage global configurations.
- **Usage Monitoring**: Admins can view detailed usage statistics for all peers and the server itself.

### Normal Peers

- **Limited Access**: Normal peers have restricted access, typically limited to their own settings and statistics.
- **View Configurations**: They can view and download their own VPN configuration files.
- **Usage Statistics**: Normal peers can monitor their own usage statistics but do not have access to other peers' data or overall server statistics.

## Backend

The backend of Wireguard UI is written in Go. It provides the necessary API endpoints for the frontend to interact with the Wireguard server and manage the VPN configuration.

### Main Dependencies

- **Gin**: A web framework used to create the API server.
- **MongoDB Driver**: Used to interact with MongoDB for storing peer data.

### Main Go Files

- `main.go`: Contains the main function that starts the API server and includes all the business logic for managing peers, configurations, and statistics.
- `go.mod`: Lists all the module dependencies required by the project.
  Here's an updated `README.md` that includes the correct installation process for both the Wireguard VPN and the Wireguard UI:

# Wireguard UI

Wireguard UI is a web-based interface designed to simplify the management of Wireguard VPN servers. It provides a user-friendly dashboard for administrators to manage VPN peers and groups, monitor usage, and handle configurations.

## Features

- **Peer Management**: Easily add, remove, and edit VPN peers.
- **Group Management**: Organize peers into groups for simplified management.
- **Configuration Management**: Generate and download configuration files for peers directly from the UI.
- **Usage Statistics**: Monitor bandwidth usage and other statistics for the VPN server and its peers.
- **Real-time Updates**: The dashboard updates peer information in real-time, without the need for page refreshes.
- **QR Code Generation**: Generate QR codes for peer configurations, facilitating easy setup of Wireguard clients.

## Installation

### Prerequisites

Before installing Wireguard UI, you must install Wireguard and generate the necessary keys:

1. Install Wireguard on your server. The installation process varies depending on your operating system. Refer to the [official documentation](https://www.wireguard.com/install/) for instructions.

2. Generate public and private keys for your Wireguard server:

   ```bash
   wg genkey | tee privatekey | wg pubkey > publickey
   ```

3. Create a Wireguard configuration file in `/etc/wireguard/`. Use the generated keys to set up your Wireguard server configuration.

4. Create a `config.json` file for Wireguard UI with the necessary details, including the paths to your Wireguard configuration and keys. You need to provide several key pieces of information that the application will use to configure its connection to MongoDB, set up the Wireguard interface, and define other operational parameters. Below is a detailed description of each field you need to include in your `config.json` file, along with an example:

```json
{
  "mongoURI": "mongodb+srv://<username>:<password>@<cluster-address>/<options>",
  "dbName": "<database-name>",
  "collectionName": "<collection-name>",
  "interfaceName": "<wireguard-interface-name>",
  "serverEndpoint": "<server-endpoint>",
  "serverPublicKey": "<server-public-key>",
  "serverNetworkAddress": "<server-network-address>",
  "path": "<path-to-wireguard-ui-folder>",
  "dnsServers": "<dns-servers>"
}
```

Here's what each field represents:

- `mongoURI`: The full MongoDB URI connection string, which includes the username, password, cluster address, and any connection options.
- `dbName`: The name of the MongoDB database where the application data will be stored.
- `collectionName`: The name of the MongoDB collection within the database to store peer information.
- `interfaceName`: The name of the Wireguard interface, typically something like `wg0`.
- `serverEndpoint`: The public endpoint of the Wireguard server, including the domain and port.
- `serverPublicKey`: The public key of the Wireguard server.
- `serverNetworkAddress`: The network address and subnet for the Wireguard server, in CIDR notation.
- `path`: The file system path where the wireguard-ui configuration files are located.
- `dnsServers`: A comma-separated list of DNS servers that the peers will use.

### Example `config.json`:

```json
{
  "mongoURI": "mongodb+srv://alireza:verySecurePassword@cluster0.meow.mongodb.net/?retryWrites=true&w=majority",
  "dbName": "wgdb",
  "collectionName": "peers",
  "interfaceName": "wg0",
  "serverEndpoint": "server1.bestwgvpn.com:42069",
  "serverPublicKey": "3SEIkOiXlNkUqfO5/Y5tS7CXMF26THkwseC38GbdpDg=",
  "serverNetworkAddress": "10.8.0.1/24",
  "path": "/root/wireguard-ui",
  "dnsServers": "1.1.1.1,8.8.8.8"
}
```

Make sure to replace the placeholder values with your actual configuration details. The `mongoURI`, `serverPublicKey`, and other sensitive information should be kept secure and not shared publicly. Save this file as `config.json` in the root directory of your Wireguard UI project or in the location specified by the application documentation.

### Installing Wireguard UI

After setting up Wireguard, proceed with the installation of Wireguard UI:

1. Clone the repository:
   ```bash
   git clone https://github.com/alirezasn3/wireguard-ui.git
   ```
2. Navigate to the project directory:
   ```bash
   cd wireguard-ui
   ```
3. Install the frontend dependencies:
   ```bash
   npm install
   ```
   or if you prefer Yarn:
   ```bash
   yarn
   ```
4. Build the frontend:
   ```bash
   npm run build
   ```
   or with Yarn:
   ```bash
   yarn build
   ```
5. Build the backend (assuming Go is installed):
   ```bash
   go build -o wireguard-ui
   ```
6. Start the Wireguard UI server:
   ```bash
   ./wireguard-ui
   ```

Now you can access the Wireguard UI dashboard through your web browser.

Here's a short description for the README about how peers are invalidated:

## Peer Invalidating Process

The Wireguard UI server periodically checks the status of all configured peers to determine if any have expired or exceeded their allowed data usage. If a peer is found to have an expiration timestamp that has passed or has used more data than permitted, the server takes steps to invalidate that peer's access.

The invalidation process involves the following steps:

1. **Generate an Invalid Preshared Key**: The server creates a non-functional preshared key to replace the peer's current valid key.
2. **Replace Preshared Key**: The invalid key is written into the Wireguard configuration in place of the peer's original key.
3. **Apply Configuration Changes**: The updated configuration is applied to the Wireguard interface to enforce the invalidation.
4. **Update Database**: The peer's status is updated in the database to reflect that they are suspended, preventing further access to the VPN.

This mechanism ensures that only active and compliant peers maintain access to the VPN, enhancing security and managing resource usage effectively.
