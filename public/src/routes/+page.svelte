<script lang="ts">
	import '../app.css';
	import type { DashboardInfo, Peer } from '../types';
	import { fade, fly } from 'svelte/transition';
	import qr from 'qrcode';

	let peers: Peer[] = [];
	let groups: { [key: string]: Peer[] } = {};
	let dashboardInfo: DashboardInfo = {
		name: '',
		isAdmin: false,
		totalRx: 0,
		totalTx: 0,
		currentRx: 0,
		currentTx: 0
	};
	let sortBy = 'expiry';
	let sortOrder = -1;
	let serch = '';
	let currentPeer: Peer | null = null;
	let view = 'peers';
	let showCreatPeer = false;
	let newName = '';
	let newExpiry = '';
	let newAllowedUsage = '';
	let newIsAdmin = false;
	let editingCurrentPeer = false;
	let showQR = false;
	let createPeerError = '';
	let updatePeerError = '';
	let deletePeerError = '';
	let resetPeerUsageError = '';

	$: {
		peers = peers
			.filter((p) => p.name.toLowerCase().includes(''))
			.sort((a, b) => {
				if (sortBy === 'expiry') return sortOrder * (a.expiresAt >= b.expiresAt ? -1 : 1);
				if (sortBy === 'usage') return sortOrder * (a.totalUsage >= b.totalUsage ? -1 : 1);
				if (sortBy === 'name') return a.name.localeCompare(b.name);
				return sortOrder * (a.currentRx >= b.currentRx ? -1 : 1);
			});
		for (let i = 0; i < peers.length; i++) {
			const groupName = peers[i].name.split('-')[0];
			if (groups[groupName]) groups[groupName].push(peers[i]);
			else groups[groupName] = [peers[i]];
		}
	}

	setInterval(async () => {
		if (editingCurrentPeer || showCreatPeer) return;
		if (currentPeer) {
			const res = await fetch('/api/peers/' + currentPeer.name);
			if (res.status === 200) {
				if (!currentPeer) return;
				currentPeer = await res.json();
			}
		} else {
			const res = await fetch(
				import.meta.env.MODE === 'development' ? 'http://my.stats:5051/api/stats' : '/api/stats'
			);
			if (res.status === 200) {
				const data = await res.json();
				peers = Object.values(data.peers as Peer[]);
				dashboardInfo = {
					name: data.name,
					isAdmin: data.isAdmin,
					totalRx: data.totalRx,
					totalTx: data.totalTx,
					currentRx: data.currentRx,
					currentTx: data.currentTx
				};
			}
		}
	}, 1000);

	function formatSeconds(totalSeconds: number, noPrefix = false) {
		if (!totalSeconds) return 'unknown';
		totalSeconds = Math.trunc(totalSeconds - Date.now() / 1000);
		const prefix = totalSeconds < 0 && !noPrefix ? '-' : '';
		totalSeconds = Math.abs(totalSeconds);
		if (totalSeconds / 60 < 1) return `${prefix}${totalSeconds} seconds`;
		const totalMinutes = Math.trunc(totalSeconds / 60);
		if (totalMinutes / 60 < 1) return `${prefix}${totalMinutes} minutes`;
		const totalHours = Math.trunc(totalMinutes / 60);
		if (totalHours / 24 < 1) return `${prefix}${totalHours} hours`;
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

	async function createPeer(name: string, isAdmin: boolean = false) {
		try {
			const res = await fetch('/api/peers/' + name, {
				method: 'POST',
				body: JSON.stringify({ isAdmin })
			});
			if (res.status === 201) {
				const data = await res.json();
				showCreatPeer = false;
				setTimeout(() => (currentPeer = data), 200);
			} else if (res.status === 400) {
				const { error } = await res.json();
				createPeerError = error;
			} else {
				createPeerError = res.status.toString();
			}
		} catch (error) {
			console.log(error);
			createPeerError = (error as Error).message;
		}
		newIsAdmin = false;
	}

	async function deletePeer(name: string) {
		try {
			const res = await fetch('/api/peers/' + name, { method: 'DELETE' });
			if (res.status === 200) {
				currentPeer = null;
				showQR = false;
				editingCurrentPeer = false;
				document.body.style.overflowY = 'auto';
			} else {
				deletePeerError = res.status.toString();
			}
		} catch (error) {
			console.log(error);
			deletePeerError = (error as Error).message;
		}
	}

	async function updatePeer(
		name: string,
		newName: string | undefined,
		newExpiry: number | undefined,
		newAllowedUsage: number | undefined
	) {
		try {
			const res = await fetch('/api/peers/' + name, {
				method: 'PATCH',
				body: JSON.stringify({ name: newName, expiresAt: newExpiry, allowedUsage: newAllowedUsage })
			});
			if (res.status === 200) {
				editingCurrentPeer = false;
				currentPeer = null;
				showQR = false;
				document.body.style.overflowY = 'auto';
			} else updatePeerError = res.status.toString();
		} catch (error) {
			console.log(error);
			updatePeerError = (error as Error).message;
		}
	}

	async function resetPeerUsage(name: string) {
		try {
			const res = await fetch('/api/peers/' + name, {
				method: 'PATCH',
				body: JSON.stringify({ totalUsage: 0 })
			});
			if (res.status === 200) {
				editingCurrentPeer = false;
				currentPeer = null;
				showQR = false;
				document.body.style.overflowY = 'auto';
			} else resetPeerUsageError = res.status.toString();
		} catch (error) {
			console.log(error);
			resetPeerUsageError = (error as Error).message;
		}
	}

	async function getConfig(name: string) {
		try {
			const res = await fetch('/api/configs/' + name);
			if (res.status === 200) {
				const config = await res.text();
				return config;
			}
		} catch (error) {
			console.log(error);
		}
	}
</script>

<nav
	class="fixed left-0 top-0 flex h-16 w-full items-center justify-between border-b-2 border-slate-900 bg-slate-950 p-4 text-lg font-bold"
>
	<span>Wireguard UI</span>
	<span>{dashboardInfo.name}</span>
</nav>
<div class="mt-16">
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
		{#if currentPeer === null}
			<button
				on:click={() => {
					newName = '';
					showCreatPeer = true;
				}}
				class="fixed bottom-6 right-6 flex h-12 w-12 items-center justify-center rounded-2xl bg-teal-700 text-lg font-bold hover:cursor-pointer hover:bg-teal-600"
			>
				<span class="absolute h-1 w-6 bg-white" />
				<span class="absolute h-1 w-6 rotate-90 bg-white" />
			</button>
		{/if}
	{/if}

	{#if peers.length}
		{#if view === 'peers'}
			<div class="overflow-y-auto md:m-4">
				<table
					class="w-full table-auto break-keep bg-slate-900 text-left max-md:text-xs md:rounded-lg"
				>
					<thead class="border-b-2 border-slate-800">
						<tr class="select-none">
							<th class="p-2 {!dashboardInfo.isAdmin&& "hidden"}">#</th>
							<th
								on:click={() => {
									sortBy = 'name';
								}}
								class="p-2 hover:cursor-pointer hover:underline {sortBy === 'name' &&
									'bg-gray-950 font-black'}">Name</th
							>
							<th
								on:click={() => {
									if (sortBy == 'expiry') {
										if (sortOrder < 0) sortOrder = 1;
										else sortOrder = -1;
									}
									sortBy = 'expiry';
								}}
								class="p-2 hover:cursor-pointer hover:underline {sortBy === 'expiry' &&
									'bg-gray-950 font-black'}">Expiry</th
							>
							<th
								on:click={() => {
									if (sortBy == 'bandwidth') {
										if (sortOrder < 0) sortOrder = 1;
										else sortOrder = -1;
									}
									sortBy = 'bandwidth';
								}}
								class="p-2 hover:cursor-pointer hover:underline {sortBy === 'bandwidth' &&
									'bg-gray-950 font-black'} {!dashboardInfo.isAdmin&& "hidden"}">Bandwidth</th
							>
							{#if dashboardInfo.isAdmin}
							<th
							on:click={() => {
									if (sortBy == 'usage') {
										if (sortOrder < 0) sortOrder = 1;
										else sortOrder = -1;
									}
									sortBy = 'usage';
								}}
								class="p-2 hover:cursor-pointer hover:underline {sortBy === 'usage' &&
									'bg-gray-950 font-black'}"
							>
							Usage</th
							>
							<th class="p-2">Allowed Usage</th>
							{:else}
								<th class="p-2">Usage</th>
							{/if}
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
								<td class="px-2 py-1 max-md:py-2 {!dashboardInfo.isAdmin&& "hidden"}">{i + 1}</td>
								<td class="px-2 py-1 max-md:py-2 {sortBy === 'name' && 'bg-gray-950 font-black'}"
									>{peer.name}</td
								>
								<td class="px-2 py-1 max-md:py-2 {sortBy === 'expiry' && 'bg-gray-950 font-black'}">
									{formatSeconds(peer.expiresAt)}
								</td>
								<td
									class="px-2 py-1 max-md:py-2 {sortBy === 'bandwidth' && 'bg-gray-950 font-black'} {!dashboardInfo.isAdmin&& "hidden"}"
									>{formatBytes(peer.currentRx)}</td
								>
								{#if dashboardInfo.isAdmin}
								<td class="px-2 py-1 max-md:py-2 {sortBy === 'usage' && 'bg-gray-950 font-black'}"
								>{formatBytes(peer.totalUsage)}</td
								>
								<td class="px-2 py-1 max-md:py-2">{formatBytes(peer.allowedUsage)}</td>
								{:else}
								<td class="px-2 py-1 max-md:py-2">{formatBytes(peer.totalUsage)} / {formatBytes(peer.allowedUsage)}</td>
								{/if}
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		{:else}
			<div />
		{/if}
	{:else}
		<div class="flex h-[calc(100vh-64px)] w-full items-center justify-center text-lg font-bold">
			Loading...
		</div>
	{/if}

	{#if currentPeer !== null}
		<div
			transition:fade={{ duration: 200 }}
			class="fixed left-0 top-16 flex h-[calc(100vh-64px)] w-[100vw] items-center justify-center bg-slate-950 bg-opacity-95 p-4 pb-0 max-md:px-0 max-md:pt-4"
		>
			<div
				transition:fly={{ y: 200, duration: 200 }}
				class="h-full w-full overflow-y-auto rounded-lg bg-slate-900 max-md:pb-16"
			>
				<div class="flex items-center justify-between rounded-t-lg bg-slate-800 px-8 py-2">
					<div class="text-2xl font-black">{currentPeer.name}</div>
					<button
						on:click={() => {
							currentPeer = null;
							showQR = false;
							editingCurrentPeer = false;
							document.body.style.overflowY = 'auto';
						}}
						class="relative h-12 w-12 hover:cursor-pointer"
					>
						<span class="absolute h-1 w-8 rotate-45 rounded bg-white" />
						<span class="absolute h-1 w-8 -rotate-45 rounded bg-white" />
					</button>
				</div>
				<div class="flex flex-col p-4">
					{#if editingCurrentPeer}
						<button
							on:click={() => (editingCurrentPeer = false)}
							class="relative mb-8 mt-2 w-fit hover:cursor-pointer"
						>
							<span class="absolute h-1 w-4 origin-left -rotate-45 rounded bg-white" />
							<span class="absolute h-1 w-6 -translate-x-0.5 rounded bg-white" />
							<span class="absolute h-1 w-4 origin-left rotate-45 rounded bg-white" />
						</button>
						<div class="mb-2">Peer's Name</div>
						<div class="mb-4 w-full">
							<input type="text" bind:value={newName} class="w-full rounded px-2 py-1 text-black" />
						</div>
						<div class="mb-2">Peer's Expiry</div>
						<div class="mb-4 flex w-full items-center">
							<input
								type="text"
								bind:value={newExpiry}
								class="w-full rounded-l px-2 py-1 text-black outline-none"
							/>
							<div class="rounded-r bg-white px-2 py-1 text-black">days</div>
						</div>
						<div class="mb-2">Peer's Usage</div>
						<div class="mb-4 flex w-full items-center">
							<input
								type="text"
								bind:value={newAllowedUsage}
								class="w-full rounded-l px-2 py-1 text-black outline-none"
							/>
							<div class="rounded-r bg-white px-2 py-1 text-black">GB</div>
						</div>
						<button
							on:click={async () => {
								if (currentPeer)
									await updatePeer(
										currentPeer.name,
										newName !== currentPeer.name ? newName : undefined,
										Math.trunc(Date.now() / 1000 + Number(newExpiry) * 3600 * 24) !==
											currentPeer.expiresAt
											? Math.trunc(Date.now() / 1000 + Number(newExpiry) * 3600 * 24)
											: undefined,
										Number(newAllowedUsage) * 1024000000 !== currentPeer.allowedUsage
											? Number(newAllowedUsage) * 1024000000
											: undefined
									);
							}}
							class="mb-4 ml-auto rounded bg-green-500 px-2 py-1 font-bold">SAVE</button
						>
						{#if updatePeerError !== ''}
							<div class="text-bold text-red-500">{updatePeerError}</div>
						{/if}
					{:else}
						<div class="mb-2 flex justify-end break-keep border-slate-700 max-md:text-sm">
							{#if dashboardInfo.isAdmin}
							<button
								on:click={() => deletePeer(currentPeer?.name || '')}
								class="ml-2 rounded-full p-4 bg-red-500 font-bold max-md:text-sm"
								><img class="h-7 w-7" src="/delete.png" alt="delete" /></button
							>
								<button
									on:click={() => {
										if (currentPeer) {
											newExpiry = (((currentPeer.expiresAt || 0) - Date.now() / 1000) / (3600 * 24))
												.toFixed(2)
												.toString();
											newAllowedUsage = Math.trunc(
												currentPeer.allowedUsage / 1024000000
											).toString();
											newName = currentPeer.name;
										}
										editingCurrentPeer = true;
									}}
									class="ml-2 rounded-full p-4 bg-orange-500 font-bold max-md:text-sm"
									><img class="h-7 w-7" src="/edit.png" alt="edit" /></button
								>
								<button
									on:click={() => resetPeerUsage(currentPeer?.name || '')}
									class="ml-2 rounded-full p-4 bg-orange-500 font-bold max-md:text-sm"
									><img class="h-7 w-7" src="/reset.png" alt="reset" /></button
								>
							{/if}
							<button
								on:click={async () => {
									const config = await getConfig(currentPeer?.name || '');
									qr.toCanvas(document.getElementById('qr-canvas'), config);
									showQR = true;
								}}
								class="ml-2 rounded-full p-4 bg-green-500 font-bold max-md:text-sm"
								><img class="h-7 w-7" src="/qr.png" alt="qrcode" /></button
							>
							<button
								on:click={async () => {
									const config = await getConfig(currentPeer?.name || '');
									const file = new Blob([config || ''], { type: 'application/octet-stream' });
									const a = document.createElement('a');
									a.href = URL.createObjectURL(file);
									a.download = currentPeer?.name.replaceAll('-', '') + '.conf';
									a.click();
								}}
								class="ml-2 rounded-full p-4 bg-green-500 font-bold max-md:text-sm"
								><img class="h-7 w-7" src="download.png" alt="download" /></button
							>
						</div>
						{#if deletePeerError}
							<div class="mb-2 text-red-500">{deletePeerError}</div>
						{/if}
						{#if resetPeerUsageError}
							<div class="mb-2 text-red-500">{resetPeerUsageError}</div>
						{/if}
						<div class="mb-2 {!dashboardInfo.isAdmin && "hidden"}">
							<div class="font-bold">Address:</div>
							<div class="ml-4 text-sm text-slate-300">{currentPeer.address}</div>
						</div>
						<div class="mb-2">
							<div class="font-bold">Usage:</div>
							<div class="ml-4 text-sm text-slate-300">
								{formatBytes(currentPeer.totalUsage)} / {formatBytes(currentPeer.allowedUsage)}
							</div>
						</div>
						<div class="mb-2 {!dashboardInfo.isAdmin && "hidden"}">
							<div class="font-bold">Bandwidth:</div>
							<div class="">
								<div class="ml-4 text-sm text-slate-300">
									<span class="text-lg">↓</span>
									{formatBytes(currentPeer.currentRx)}
								</div>
								<div class="ml-4 text-sm text-slate-300">
									<span class="text-lg">↑</span>
									{formatBytes(currentPeer.currentTx)}
								</div>
							</div>
						</div>
						<div class="mb-2">
							<div class="font-bold">Latest Handshake:</div>
							<div class="ml-4 text-sm text-slate-300">
								{formatSeconds(currentPeer.latestHandshake, true)} ago
							</div>
						</div>
						<div class="mb-2">
							<div class="font-bold">Expiry:</div>
							<div class="ml-4 text-sm text-slate-300">{formatSeconds(currentPeer.expiresAt)}</div>
						</div>
						<canvas
							class="max-md:w-[calc(100vw-64)] {showQR ? 'max-h-fit' : 'max-h-0'}"
							id="qr-canvas"
						/>
					{/if}
				</div>
			</div>
		</div>
	{/if}

	{#if showCreatPeer}
		<div
			transition:fade={{ duration: 200 }}
			class="fixed left-0 top-16 flex h-[calc(100vh-64px)] w-[100vw] items-center justify-center bg-slate-950 bg-opacity-95 p-4 pb-0 max-md:px-0 max-md:pt-4"
		>
			<div
				transition:fly={{ y: 200, duration: 200 }}
				class="h-full w-full overflow-y-auto rounded-lg bg-slate-900 max-md:pb-16"
			>
				<div class="flex items-center justify-between border-b-2 border-slate-800 px-8 py-2">
					<div class="text-2xl font-black">Crete Peer</div>
					<button
						on:click={() => {
							showCreatPeer = false;
							newIsAdmin = false;
							document.body.style.overflowY = 'auto';
						}}
						class="relative h-12 w-12 hover:cursor-pointer"
					>
						<span class="absolute h-1 w-8 rotate-45 rounded bg-white" />
						<span class="absolute h-1 w-8 -rotate-45 rounded bg-white" />
					</button>
				</div>
				<div class="flex flex-col p-4">
					<div class="mb-2">Peer's Name</div>
					<div class="mb-4 w-full">
						<input type="text" bind:value={newName} class="w-full rounded px-2 py-1 text-black" />
					</div>
					<div class="mb-4 flex items-center">
						<input bind:checked={newIsAdmin} type="checkbox" name="isAdmin" id="isAdmin" />
						<label for="isAdmin" class="ml-1">Is Admin</label>
					</div>
					<button
						on:click={async () => {
							await createPeer(newName, newIsAdmin);
							if (createPeerError === '') {
								showCreatPeer = false;
							}
						}}
						class="mb-4 ml-auto rounded bg-green-500 px-2 py-1 font-bold">CREATE</button
					>
					{#if createPeerError !== ''}
						<div class="text-bold text-red-500">{createPeerError}</div>
					{/if}
				</div>
			</div>
		</div>
	{/if}
</div>
