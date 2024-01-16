export interface DashboardInfo {
	name: string;
	role: string;
}

export interface Peer {
	name: string;
	latestHandshake: number;
	address: string;
	expiresAt: number;
	currentRx: number;
	currentTx: number;
	allowedUsage: number;
	totalUsage: number;
	presharedKey: string;
	publicKey: string;
	role: string;
}
