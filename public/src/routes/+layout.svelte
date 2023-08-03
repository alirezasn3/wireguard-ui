<script lang="ts">
	import '../app.css';
	import DashboardInfoStore from '../store';
	import type { DashboardInfo } from '../types';

	let dashboardInfo: DashboardInfo;
	DashboardInfoStore.subscribe((info) => (dashboardInfo = info));
</script>

<nav
	class="fixed left-0 top-0 flex h-16 w-full items-center justify-between border-b-2 border-slate-900 bg-slate-950 p-4 text-lg font-bold"
>
	<span>Wireguard UI</span>
	<span>{dashboardInfo.name}</span>
</nav>

<div class="mt-16">
	<slot />
</div>
