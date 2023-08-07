export interface DashboardInfo {
	name: string;
	isAdmin: boolean;
	totalRx: number;
	totalTx: number;
	currentRx: number;
	currentTx: number;
}

export interface Peer {
	name: string;
	latestHandshake: number;
	address: string;
	expiresAt: number;
	currentRx: number;
	currentTx: number;
	allowedUsage: number;
	remainingUsage: number;
	presharedKey: string;
	publicKey: string;
	config: string;
}
