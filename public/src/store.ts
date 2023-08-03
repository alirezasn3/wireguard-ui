import { writable } from 'svelte/store';
import type { DashboardInfo } from './types';

const DashboardInfoStore = writable<DashboardInfo>({} as DashboardInfo);

export default DashboardInfoStore;
