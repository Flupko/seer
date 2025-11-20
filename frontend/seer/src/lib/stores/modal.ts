// stores/modalStore.ts
import { create } from 'zustand';

export type ModalType = "auth" | "profileCompletion" | "changePassword" | "setPassword" | "user" | "bet" | "betSuccess" | null;

interface ModalStore {
    currentModal: ModalType;
    modalData: Record<string, any> | null;
    openModal: (modal: ModalType, data?: any) => void;
    closeModal: () => void;
}

export const useModalStore = create<ModalStore>((set) => ({
    currentModal: null,
    modalData: null,

    openModal: (modal, data) => set({
        currentModal: modal,
        modalData: data || null
    }),

    closeModal: () => set({
        currentModal: null,
        modalData: null
    }),
}));
