// src/contexts/ModalContext.tsx
"use client";
import { ModalType } from "@/lib/stores/modal";
import AuthModal from "@/ui/auth/AuthModal";
import { useMediaQuery } from "usehooks-ts";
import ProfileCompletionModal from "../auth/ProfileCompletionModal";
import ChangePasswordModal from "../profile/ChangePasswordModal";
import SetPasswordModal from "../profile/SetPasswordModal";
import ModalDesktop from "./ModalDesktop";
import ModalMobile from "./ModalMobile";



export const modalComponents: Record<Exclude<ModalType, null>, { content: React.ComponentType<any>; height: string; desktopWidth: string }> = {
  auth: { content: AuthModal, height: "h-[720px]", desktopWidth: "max-w-lg" },
  profileCompletion: { content: ProfileCompletionModal, height: "", desktopWidth: "max-w-[30.5rem]" },
  changePassword: { content: ChangePasswordModal, height: "", desktopWidth: "max-w-[30.5rem]" },
  setPassword: { content: SetPasswordModal, height: "", desktopWidth: "max-w-[30.5rem]" },
};

export function ModalContainer() {

  const isMobile = useMediaQuery('(max-width: 768px)');

  return (
    <div>
      {isMobile ? <ModalMobile /> : <ModalDesktop />}
    </div>

  );
}