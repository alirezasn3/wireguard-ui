export interface DashboardInfo {
	name: string;
	isAdmin: boolean;
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
}
