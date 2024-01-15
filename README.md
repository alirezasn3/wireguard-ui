Here's an updated `README.md` with a new section that explains the differences between admin and normal peers within the Wireguard UI:

```markdown
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

```markdown
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

4. Create a `config.json` file for Wireguard UI with the necessary details, including the paths to your Wireguard configuration and keys.

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

## Contributing

Contributions are welcome! Please feel free to submit a pull request or open an issue if you have feedback or suggestions.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- Thanks to the creators and maintainers of Wireguard for the fantastic VPN software.
- Thanks to all contributors of the `alirezasn3/wireguard-ui` project.
```

This section provides a clear distinction between the roles and capabilities of admin and normal peers within the Wireguard UI, which should help users understand the level of access and control they have based on their role. Adjustments can be made as necessary to fit the specifics of the project and its documentation.