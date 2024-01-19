<script lang="ts">
	import '../app.css';
	import type { DashboardInfo, Peer } from '../types';
	import { fade, fly } from 'svelte/transition';
	import qr from 'qrcode';
	import { onMount } from 'svelte';

	let peers: Peer[] = [];
	let groups: { [key: string]: Peer[] } = {};
	let dashboardInfo: DashboardInfo = {
		name: '',
		role: 'user'
	};
	let sortBy = 'expiry';
	let sortOrder = -1;
	let currentPeer: Peer | null = null;
	let view = 'peers';
	let showCreatPeer = false;
	let newName = '';
	let newExpiry = '';
	let newAllowedUsage = '';
	let newRole = 'user';
	let editingCurrentPeer = false;
	let createPeerError = '';
	let updatePeerError = '';
	let deletePeerError = '';
	let resetPeerUsageError = '';
	let search = '';

	$: {
		if (view === 'peers') {
			peers = peers
				.filter((p) => p.name.toLowerCase().includes(search.toLocaleLowerCase()))
				.sort((a, b) => {
					if (sortBy === 'expiry') return sortOrder * (a.expiresAt >= b.expiresAt ? -1 : 1);
					if (sortBy === 'usage') return sortOrder * (a.totalUsage >= b.totalUsage ? -1 : 1);
					if (sortBy === 'name') return a.name.localeCompare(b.name);
					return sortOrder * (a.currentRx >= b.currentRx ? -1 : 1);
				});
		} else {
			groups = {};
			for (let i = 0; i < peers.length; i++) {
				const groupName = peers[i].name.split('-')[0];
				if (groups[groupName]) groups[groupName].push(peers[i]);
				else groups[groupName] = [peers[i]];
			}
		}
	}

	onMount(async () => {
		var ws = new WebSocket(
			(window.location.protocol === 'https:' ? 'wss://' : 'ws://') + window.location.host + '/ws'
		);
		ws.onopen = () => {
			console.log('ws opend');
		};
		ws.onmessage = ({ data }) => {
			if (editingCurrentPeer || showCreatPeer) return;
			const parsedData = JSON.parse(data);
			if (currentPeer) {
				currentPeer = parsedData.peers[currentPeer.publicKey];
			} else {
				peers = Object.values(parsedData.peers as Peer[]);
				dashboardInfo = {
					name: parsedData.name,
					role: parsedData.role
				};
			}
		};
		ws.onclose = () => {
			console.log('ws closed');
		};
	});

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

	function formatBytes(totalBytes: number, space = true) {
		if (!totalBytes) return `00.00${space ? ' ' : ''}KB`;
		const totalKilos = totalBytes / 1024;
		const totalMegas = totalKilos / 1000;
		const totalGigas = totalMegas / 1000;
		const totalTeras = totalGigas / 1000;
		if (totalKilos < 100)
			return `${totalKilos < 10 ? '0' : ''}${totalKilos.toFixed(2)}${space ? ' ' : ''}KB`;
		if (totalMegas < 100)
			return `${totalMegas < 10 ? '0' : ''}${totalMegas.toFixed(2)}${space ? ' ' : ''}MB`;
		if (totalGigas < 100)
			return `${totalGigas < 10 ? '0' : ''}${totalGigas.toFixed(2)}${space ? ' ' : ''}GB`;
		return `${totalTeras < 10 ? '0' : ''}${totalTeras.toFixed(2)}${space ? ' ' : ''}TB`;
	}

	async function createPeer(name: string, role: string) {
		try {
			const res = await fetch('/api/peers/' + name, {
				method: 'POST',
				body: JSON.stringify({ role: role || 'user' })
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
		newRole = 'user';
	}

	async function deletePeer(name: string) {
		try {
			const res = await fetch('/api/peers/' + name, { method: 'DELETE' });
			if (res.status === 200) {
				currentPeer = null;
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
		newAllowedUsage: number | undefined,
		newRole: string | undefined
	) {
		try {
			if (name === newName) newName = undefined;
			const res = await fetch('/api/peers/' + name, {
				method: 'PATCH',
				body: JSON.stringify({
					name: newName,
					expiresAt: newExpiry,
					allowedUsage: newAllowedUsage,
					role: newRole
				})
			});
			if (res.status === 200) {
				if (newName && currentPeer) currentPeer.name = newName;
				editingCurrentPeer = false;
			} else updatePeerError = res.status.toString();
		} catch (error) {
			console.log(error);
			updatePeerError = (error as Error).message;
		}
	}

	async function resetPeerUsage(name: string) {
		try {
			const res = await fetch('/api/reset-usage/' + name);
			if (res.status === 200) {
				editingCurrentPeer = false;
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

	function dataURLtoFile(dataurl: string, filename: string, type: string) {
		let arr = dataurl.split(',');
		let bstr = atob(arr[arr.length - 1]);
		let n = bstr.length;
		let u8arr = new Uint8Array(n);
		while (n--) {
			u8arr[n] = bstr.charCodeAt(n);
		}
		return new File([u8arr], filename, { type });
	}

	async function share(token: string, config: string, name: string) {
		try {
			const dataurl = await qr.toDataURL(
				document.getElementById('qr-canvas') as HTMLCanvasElement,
				config
			);
			await navigator.share({
				title: name,
				url: `https://t.me/wgcrocbot?start=${token}`,
				files: [dataURLtoFile(dataurl, `${name}.png`, 'image/png')]
			});
		} catch (error) {
			console.log(error);
		}
	}
</script>

<nav
	class="fixed left-0 top-0 flex h-16 w-full items-center justify-between border-b-2 border-slate-900 bg-slate-950 p-4 text-lg font-bold"
>
	<span>Wireguard UI</span>
	<span class="text-sm">{dashboardInfo.name}</span>
</nav>
<div class="mt-16">
	{#if dashboardInfo.role === 'admin' || dashboardInfo.role === 'distributor'}
		<div class="mx-8 my-4 flex items-center justify-between pt-4 max-md:mx-4 max-md:text-sm">
			<div>{peers.length} Peers</div>
			{#if dashboardInfo.role === 'admin'}
				<div>{Object.keys(groups).length} Groups</div>
			{/if}
		</div>
		<div class="w-full px-4">
			<input
				placeholder="search peers"
				type="text"
				class="w-full rounded p-2 font-bold text-slate-50 text-slate-950"
				bind:value={search}
			/>
		</div>
		<div class="m-4 flex">
			<button
				on:click={() => {
					newName = '';
					showCreatPeer = true;
					document.body.style.overflowY = 'hidden';
				}}
				class="mr-2 flex items-center justify-center rounded bg-green-500 px-2 py-1 text-lg font-bold hover:cursor-pointer hover:bg-green-600"
			>
				ADD PEER
			</button>
			{#if dashboardInfo.role === 'admin'}
				<button
					class="flex items-center justify-center rounded bg-orange-500 px-2 py-1 text-lg font-bold hover:cursor-pointer hover:bg-orange-600"
					on:click={() => {
						if (view === 'peers') view = 'groups';
						else view = 'peers';
					}}>SHOW {view === 'peers' ? 'GROUPS' : 'PEERS'}</button
				>
			{/if}
		</div>
	{/if}

	{#if peers.length}
		<div class="m-4 overflow-y-auto">
			{#if view === 'peers'}
				<table
					class="w-full table-auto break-keep bg-slate-900 text-left max-md:text-xs md:rounded-lg"
				>
					<thead class="border-b-2 border-slate-800">
						<tr class="select-none">
							<th class="p-2 {dashboardInfo.role !== 'admin' && 'hidden'}">#</th>
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
									'bg-gray-950 font-black'} {dashboardInfo.role !== 'admin' && 'hidden'}"
								>Bandwidth</th
							>
							{#if dashboardInfo.role === 'admin'}
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
								on:click={async () => {
									currentPeer = peer;
									document.body.style.overflowY = 'hidden';
									createPeerError = '';
									updatePeerError = '';
									deletePeerError = '';
									resetPeerUsageError = '';
									const config = await getConfig(currentPeer?.name || '');
									qr.toCanvas(document.getElementById('qr-canvas'), config || '');
								}}
								class="hover:bg-slate-800"
							>
								<td class="px-2 py-1 max-md:py-2 {dashboardInfo.role !== 'admin' && 'hidden'}"
									>{i + 1}</td
								>
								<td
									class="whitespace-nowrap px-2 py-1 max-md:py-2 {sortBy === 'name' &&
										'bg-gray-950 font-black'}">{peer.name}</td
								>
								<td
									class="whitespace-nowrap px-2 py-1 max-md:py-2 {sortBy === 'expiry' &&
										'bg-gray-950 font-black'} {Math.trunc(peer.expiresAt - Date.now() / 1000) < 0 &&
										'text-red-500'}"
								>
									{formatSeconds(peer.expiresAt)}
								</td>
								<td
									class="whitespace-nowrap px-2 py-1 max-md:py-2 {sortBy === 'bandwidth' &&
										'bg-gray-950 font-black'} {dashboardInfo.role !== 'admin' && 'hidden'}"
									>{formatBytes(peer.currentRx)}</td
								>
								{#if dashboardInfo.role === 'admin'}
									<td
										class="whitespace-nowrap px-2 py-1 max-md:py-2 {sortBy === 'usage' &&
											'bg-gray-950 font-black'} {peer.totalUsage >= peer.allowedUsage &&
											'text-red-500'}"
										>{formatBytes(peer.totalUsage)} / {formatBytes(peer.allowedUsage, false)}</td
									>
								{:else}
									<td
										class="whitespace-nowrap px-2 py-1 max-md:py-2 {peer.totalUsage >=
											peer.allowedUsage && 'text-red-500'}"
										>{formatBytes(peer.totalUsage)} / {formatBytes(peer.allowedUsage, false)}</td
									>
								{/if}
							</tr>
						{/each}
					</tbody>
				</table>
			{:else}
				{#each Object.keys(groups) as groupName}
					<table
						class="mb-4 w-full table-auto break-keep bg-slate-900 text-left max-md:text-xs md:rounded-lg"
					>
						<tbody
							class="hover:cursor-pointer [&>*:nth-child(even)]:border-y [&>*:nth-child(even)]:border-slate-800"
						>
							{#each groups[groupName] as peer, i}
								<tr
									on:click={() => {
										currentPeer = peer;
										document.body.style.overflowY = 'hidden';
									}}
									class="hover:bg-slate-800"
								>
									<td class="px-2 py-1 max-md:py-2">{i + 1}</td>
									<td class="whitespace-nowrap px-2 py-1 max-md:py-2">{peer.name}</td>
									<td
										class="whitespace-nowrap px-2 py-1 max-md:py-2 {Math.trunc(
											peer.expiresAt - Date.now() / 1000
										) < 0 && 'text-red-500'}"
									>
										{formatSeconds(peer.expiresAt)}
									</td>
									<td class="whitespace-nowrap px-2 py-1 max-md:py-2"
										>{formatBytes(peer.currentRx)}</td
									>
									<td
										class="whitespace-nowrap px-2 py-1 max-md:py-2 {peer.totalUsage >=
											peer.allowedUsage && 'text-red-500'}"
										>{formatBytes(peer.totalUsage)} / {formatBytes(peer.allowedUsage)}</td
									>
								</tr>
							{/each}
						</tbody>
					</table>
				{/each}
			{/if}
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
							editingCurrentPeer = false;
							newRole = '';
							newName = '';
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
						<div class="mb-4 flex items-center">
							{#if dashboardInfo.role === 'distributor'}
								<div class="mr-1">{dashboardInfo.name.split('-')[0]}-</div>
							{/if}
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
						{#if dashboardInfo.role === 'admin'}
							<label for="role" class="mb-2 text-white">Role</label>
							<select
								bind:value={newRole}
								name="role"
								id="role"
								class="mb-4 rounded px-2 py-1 text-black"
							>
								<option value="user">User</option>
								<option value="distributor">Distributor</option>
								<option value="admin">Admin</option>
							</select>
						{/if}
						<button
							on:click={async () => {
								if (currentPeer)
									await updatePeer(
										currentPeer.name,
										dashboardInfo.role === 'admin'
											? newName
											: `${dashboardInfo.name.split('-')[0]}-${newName}`,
										Math.trunc(Date.now() / 1000 + Number(newExpiry) * 3600 * 24) !==
											currentPeer.expiresAt
											? Math.trunc(Date.now() / 1000 + Number(newExpiry) * 3600 * 24)
											: undefined,
										Number(newAllowedUsage) * 1024000000 !== currentPeer.allowedUsage
											? Number(newAllowedUsage) * 1024000000
											: undefined,
										newRole !== currentPeer.role ? newRole : undefined
									);
							}}
							class="mb-4 ml-auto rounded bg-green-500 px-2 py-1 font-bold">SAVE</button
						>
						{#if updatePeerError !== ''}
							<div class="text-bold text-red-500">{updatePeerError}</div>
						{/if}
					{:else}
						<div class="mb-2 flex justify-end break-keep border-slate-700 max-md:text-sm">
							{#if dashboardInfo.role !== 'user'}
								<button
									on:click={() => deletePeer(currentPeer?.name || '')}
									class="ml-2 rounded-full bg-red-500 p-2 font-bold max-md:text-sm"
									><img class="h-6 w-6 invert" src="/delete.png" alt="delete" /></button
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
											if (dashboardInfo.role === 'admin') newName = currentPeer.name;
											else newName = currentPeer.name.split('-').slice(1).join('-');
											newRole = currentPeer.role;
										}
										editingCurrentPeer = true;
									}}
									class="ml-2 rounded-full bg-orange-500 p-2 font-bold max-md:text-sm"
									><img class="h-6 w-6 invert" src="/edit.png" alt="edit" /></button
								>
								<button
									on:click={() => resetPeerUsage(currentPeer?.name || '')}
									class="ml-2 rounded-full bg-orange-500 p-2 font-bold max-md:text-sm"
									><img class="h-6 w-6 invert" src="/reset.png" alt="reset" /></button
								>
								<button
									on:click={async () => {
										const config = await getConfig(currentPeer?.name || '');
										share(currentPeer?.telegramToken || '', config || '', currentPeer?.name || '');
									}}
									class="ml-2 rounded-full bg-green-500 p-2 font-bold max-md:text-sm"
									><img class="h-6 w-6 invert" src="share.png" alt="share" /></button
								>
							{/if}
							<button
								on:click={async () => {
									const config = await getConfig(currentPeer?.name || '');
									const file = new Blob([config || ''], { type: 'application/octet-stream' });
									const a = document.createElement('a');
									a.href = URL.createObjectURL(file);
									a.download = currentPeer?.name.replaceAll('-', '') + '.conf';
									a.click();
								}}
								class="ml-2 rounded-full bg-green-500 p-2 font-bold max-md:text-sm"
								><img class="h-6 w-6 invert" src="download.png" alt="download" /></button
							>
						</div>
						{#if deletePeerError}
							<div class="mb-2 text-red-500">{deletePeerError}</div>
						{/if}
						{#if resetPeerUsageError}
							<div class="mb-2 text-red-500">{resetPeerUsageError}</div>
						{/if}
						<div class="mb-2 {dashboardInfo.role !== 'admin' && 'hidden'}">
							<div class="font-bold">Address:</div>
							<div class="ml-4 text-sm text-slate-300">{currentPeer.address}</div>
						</div>
						<div class="mb-2">
							<div class="font-bold">Usage:</div>
							<div class="ml-4 text-sm text-slate-300">
								{formatBytes(currentPeer.totalUsage)} / {formatBytes(currentPeer.allowedUsage)}
							</div>
						</div>
						<div class="mb-2 {dashboardInfo.role !== 'admin' && 'hidden'}">
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
						<div class="mb-2">
							<div class="font-bold">Telegram Bot Token:</div>
							<div class="ml-4 text-sm text-slate-300">{currentPeer.telegramToken}</div>
						</div>
					{/if}
					<canvas
						class="{editingCurrentPeer && 'hidden'} max-md:w-[calc(100vw-64)]"
						id="qr-canvas"
					/>
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
							newRole = 'user';
							document.body.style.overflowY = 'auto';
						}}
						class="relative h-12 w-12 hover:cursor-pointer"
					>
						<span class="absolute h-1 w-8 rotate-45 rounded bg-white" />
						<span class="absolute h-1 w-8 -rotate-45 rounded bg-white" />
					</button>
				</div>
				<div class="flex flex-col p-4">
					<label for="name" class="mb-2">Peer's Name</label>
					<div class="mb-4 flex items-center">
						{#if dashboardInfo.role === 'distributor'}
							<div class="mr-1">{dashboardInfo.name.split('-')[0]}-</div>
						{/if}
						<input
							name="name"
							id="name"
							type="text"
							bind:value={newName}
							class="w-full rounded px-2 py-1 text-black"
						/>
					</div>
					{#if dashboardInfo.role === 'admin'}
						<label for="role" class="mb-2 text-white">Role</label>
						<select
							bind:value={newRole}
							name="role"
							id="role"
							class="mb-4 rounded px-2 py-1 text-black"
						>
							<option value="user">User</option>
							<option value="distributor">Distributor</option>
							<option value="admin">Admin</option>
						</select>
					{/if}
					<button
						on:click={async () => {
							await createPeer(
								dashboardInfo.role === 'admin'
									? newName
									: `${dashboardInfo.name.split('-')[0]}-${newName}`,
								newRole
							);
							if (createPeerError === '') {
								showCreatPeer = false;
							}
						}}
						class="ml-auto rounded bg-green-500 px-2 py-1 font-bold">CREATE</button
					>
					{#if createPeerError !== ''}
						<div class="text-bold text-red-500">{createPeerError}</div>
					{/if}
				</div>
			</div>
		</div>
	{/if}
</div>
