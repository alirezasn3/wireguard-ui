<script lang="ts">
	import DashboardInfoStore from '../store';
	import type { DashboardInfo, Peer } from '../types';

	let peers: Peer[] = [];
	let groups: { [key: string]: Peer[] } = {};
	let dashboardInfo: DashboardInfo;
	let sortBy = 'expiry';
	let sortOrder = 1;
	let serch = '';
	let currentPeer: Peer | null = null;
	let view = 'peers';

	DashboardInfoStore.subscribe((info) => (dashboardInfo = info));

	$: {
		peers = peers
			.filter((p) => p.name.toLowerCase().includes(''))
			.sort((a, b) => {
				if (sortBy === 'expiry') return sortOrder * (a.expiresAt >= b.expiresAt ? -1 : 1);
				if (sortBy === 'usage') return sortOrder * (a.totalRx >= b.totalRx ? -1 : 1);
				return sortOrder * (a.currentRx >= b.currentRx ? -1 : 1);
			});
		for (let i = 0; i < peers.length; i++) {
			const groupName = peers[i].name.split('-')[0];
			if (groups[groupName]) groups[groupName].push(peers[i]);
			else groups[groupName] = [peers[i]];
		}
	}

	setInterval(async () => {
		const res = await fetch('/api/stats');
		if (res.status === 200) {
			const data = await res.json();
			peers = Object.values(data.peers as Peer[]);
			DashboardInfoStore.set({
				name: data.name,
				isAdmin: data.isAdmin,
				totalRx: data.totalRx,
				totalTx: data.totalTx,
				currentRx: data.currentRx,
				currentTx: data.currentTx
			} as DashboardInfo);
		}
	}, 1000);

	function formatSeconds(totalSeconds: number) {
		if (!totalSeconds) return 'unknown';
		totalSeconds = Math.trunc(totalSeconds - Date.now() / 1000);
		const prefix = totalSeconds < 0 ? '-' : '';
		totalSeconds = Math.abs(totalSeconds);
		if (totalSeconds / 60 < 1) return `${prefix}${totalSeconds} seconds`;
		const totalMinutes = Math.trunc(totalSeconds / 60);
		if (totalMinutes / 60 < 1) return `${prefix}${totalMinutes} minutes`;
		const totalHours = Math.trunc(totalMinutes / 60);
		if (totalHours / 60 < 1) return `${prefix}${totalHours} hours`;
		return `${prefix}${Math.trunc(totalHours / 24)} days`;
	}

	function formatBytes(totalBytes: number) {
		if (!totalBytes) return '00.00 KB';
		const totalKilos = totalBytes / 1024;
		const totalMegas = totalKilos / 1000;
		const totalGigas = totalMegas / 1000;
		const totalTeras = totalGigas / 1000;
		if (totalKilos < 100) return `${totalKilos < 10 ? '0' : ''}${totalKilos.toFixed(2)} KB`;
		if (totalMegas < 100) return `${totalMegas < 10 ? '0' : ''}${totalMegas.toFixed(2)} MB`;
		if (totalGigas < 100) return `${totalGigas < 10 ? '0' : ''}${totalGigas.toFixed(2)} GB`;
		return `${totalTeras < 10 ? '0' : ''}${totalTeras.toFixed(2)} TB`;
	}
</script>

{#if dashboardInfo.isAdmin}
	<div class="mx-4 mt-16 border-b-2 border-slate-900 p-4 font-bold max-md:px-0 max-md:text-sm">
		<div class="flex items-center">
			<div>&#8595; {formatBytes(dashboardInfo.currentRx)}/S</div>
			<div class="mx-3 h-1.5 w-1.5 rounded-full bg-slate-700" />
			<div>{formatBytes(dashboardInfo.totalRx)}</div>
		</div>
		<div class="flex items-center">
			<div>&#8593; {formatBytes(dashboardInfo.currentTx)}/S</div>
			<div class="mx-3 h-1.5 w-1.5 rounded-full bg-slate-700" />
			<div>{formatBytes(dashboardInfo.totalTx)}</div>
		</div>
	</div>
	<div class="mx-8 my-4 flex items-center justify-between max-md:mx-4 max-md:text-sm">
		<div>{peers.length} Peers</div>
		<div>{Object.keys(groups).length} Groups</div>
	</div>
{/if}

{#if peers.length}
	{#if view === 'peers'}
		<div class="md:m-4">
			<table class="w-full table-auto bg-slate-900 text-left max-md:text-xs md:rounded-lg">
				<thead class="border-b-2 border-slate-800">
					<tr class="select-none">
						<th class="p-2">#</th>
						<th class="p-2">Name</th>
						<th
							on:click={() => {
								sortBy = 'expiry';
								if ((sortBy = 'expiry')) {
									if (sortOrder < 0) sortOrder = 1;
									else sortOrder = -1;
								}
							}}
							class="p-2 hover:cursor-pointer hover:underline {sortBy === 'expiry' &&
								'bg-gray-950 font-black'}"
							>{sortBy === 'expiry' && sortOrder === 1 ? '↑' : sortBy === 'expiry' ? '↓' : ''} Expiry</th
						>
						<th
							on:click={() => {
								sortBy = 'bandwidth';
								if ((sortBy = 'bandwidth')) {
									if (sortOrder < 0) sortOrder = 1;
									else sortOrder = -1;
								}
							}}
							class="p-2 hover:cursor-pointer hover:underline {sortBy === 'bandwidth' &&
								'bg-gray-950 font-black'}"
							>{sortBy === 'bandwidth' && sortOrder === 1 ? '↑' : sortBy === 'bandwidth' ? '↓' : ''}
							Bandwidth</th
						>
						<th
							on:click={() => {
								sortBy = 'usage';
								if ((sortBy = 'usage')) {
									if (sortOrder < 0) sortOrder = 1;
									else sortOrder = -1;
								}
							}}
							class="p-2 hover:cursor-pointer hover:underline {sortBy === 'usage' &&
								'bg-gray-950 font-black'}"
							>{sortBy === 'usage' && sortOrder === 1 ? '↑' : sortBy === 'usage' ? '↓' : ''} Usage</th
						>
					</tr>
				</thead>
				<tbody
					class="hover:cursor-pointer [&>*:nth-child(even)]:border-y-[1px] [&>*:nth-child(even)]:border-slate-800"
				>
					{#each peers as peer, i}
						<tr
							on:click={() => {
								currentPeer = peer;
								document.body.style.overflowY = 'hidden';
							}}
							class="{Math.trunc(peer.expiresAt - Date.now() / 1000) < 0 &&
								'text-red-500'} hover:bg-slate-800"
						>
							<td class="px-2 py-1">{i + 1}</td>
							<td class="px-2 py-1">{peer.name}</td>
							<td class="px-2 py-1 {sortBy === 'expiry' && 'bg-gray-950 font-black'}">
								<span class="hidden max-md:block"
									>{formatSeconds(peer.expiresAt).replace(' ', '')}</span
								>
								<span class="hidden md:block">{formatSeconds(peer.expiresAt)}</span>
							</td>
							<td class="px-2 py-1 {sortBy === 'bandwidth' && 'bg-gray-950 font-black'}"
								><span class="hidden max-md:block"
									>{formatBytes(peer.currentRx).replace(' ', '')}</span
								>
								<span class="hidden md:block">{formatBytes(peer.currentRx)}</span></td
							>
							<td class="px-2 py-1 {sortBy === 'usage' && 'bg-gray-950 font-black'}"
								><span class="hidden max-md:block"
									>{formatBytes(peer.totalRx + peer.totalTx).replace(' ', '')}</span
								>
								<span class="hidden md:block">{formatBytes(peer.totalRx + peer.totalTx)}</span></td
							>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	{:else}
		<div />
	{/if}
{:else}
	<div class="flex h-full w-full items-center justify-center text-lg font-bold">Loading...</div>
{/if}

{#if currentPeer !== null}
	<div
		class="fixed left-0 top-16 flex h-[calc(100vh-64px)] w-[100vw] items-center justify-center bg-slate-950 bg-opacity-95 p-4"
	>
		<div class="m-4 h-full max-h-[400px] w-full max-w-[400px] rounded-lg bg-slate-900">
			<div class="border-b-2 border-slate-800 p-4">
				<button
					on:click={() => {
						currentPeer = null;
						document.body.style.overflowY = 'auto';
					}}
					class="text-2xl font-black">X</button
				>
			</div>
			<div class="flex flex-col p-4">
				<span>{currentPeer.name}</span>
				<span>{currentPeer.allowedIps}</span>
				<span>{currentPeer.totalRx}</span>
				<span>{currentPeer.totalTx}</span>
				<span>{currentPeer.currentRx}</span>
				<span>{currentPeer.currentTx}</span>
				<span>{currentPeer.latestHandshake}</span>
				<span>{currentPeer.expiresAt}</span>
			</div>
		</div>
	</div>
{/if}
