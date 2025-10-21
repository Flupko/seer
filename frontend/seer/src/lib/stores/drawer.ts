// stores/modalStore.ts
import { create } from 'zustand';

export type DrawerType = "chat" | "bet" | "betSuccess" | null

interface DrawerStore {
    currentDrawer: DrawerType;
    drawerData: Record<string, any> | null;
    openDrawer: (drawer: DrawerType, data?: any) => void;
    closeDrawer: () => void;
}

export const useDrawerStore = create<DrawerStore>((set) => ({
    currentDrawer: null,
    drawerData: null,

    openDrawer: (drawer, data) => set({
        currentDrawer: drawer,
        drawerData: data || null
    }),

    closeDrawer: () => set({
        currentDrawer: null,
        drawerData: null
    }),
}
));
