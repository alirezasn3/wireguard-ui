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
	totalRx: number;
	totalTx: number;
	latestHandshake: number;
	allowedIps: string;
	expiresAt: number;
	currentRx: number;
	currentTx: number;
	presharedKey: string;
	publicKey: string;
}
