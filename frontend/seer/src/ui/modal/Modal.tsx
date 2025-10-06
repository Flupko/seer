// src/contexts/ModalContext.tsx
"use client";
import AuthModal from "@/ui/auth/AuthModal";
import { AnimatePresence, motion } from "motion/react";
import { createContext, useContext, useState, ReactNode } from "react";
import ModalDesktop from "./ModalDesktop";
import ModalMobile from "./ModalMobile";
import { useMediaQuery } from "usehooks-ts";
import ProfileCompletionModal from "../auth/ProfileCompletionModal";

type ModalType = "auth" | "profileCompletion" | null;

const ModalContext = createContext<{
  currentModal: ModalType;
  modalData: any;
  openModal: (modal: ModalType, data?: any) => void;
  closeModal: () => void;
} | null>(null);

export function ModalProvider({ children }: { children: ReactNode }) {
  const [currentModal, setCurrentModal] = useState<ModalType>(null);
  const [modalData, setModalData] = useState<any>(null);

  return (
    <ModalContext.Provider
      value={{
        currentModal,
        modalData,
        openModal: (modal, data) => {
          setCurrentModal(modal);
          setModalData(data || null);
        },
        closeModal: () => {
          setCurrentModal(null);
          setModalData(null);
        },

      }}
    >
      {children}
    </ModalContext.Provider>
  );
}

export const useModal = () => {
  const context = useContext(ModalContext);
  if (!context) throw new Error("useModal must be used within ModalProvider");
  return context;
};


export const modalComponents: Record<Exclude<ModalType, null>, { content: React.ComponentType<any>; height: string; desktopWidth: string }> = {
  auth: { content: AuthModal, height: "h-[720px]", desktopWidth: "max-w-lg" },
  profileCompletion: { content: ProfileCompletionModal, height: "", desktopWidth: "max-w-lg" },
};

export function ModalContainer() {

  const isMobile = useMediaQuery('(max-width: 768px)');

  return (
    <div>
      {isMobile  ? <ModalMobile /> : <ModalDesktop />}
    </div>

  );
}