import { useModalStore } from "@/lib/stores/modal";
import { AnimatePresence, LayoutGroup, motion, useDragControls } from "motion/react";
import { useRef } from "react";
import { modalComponents } from "./Modal";

export default function MobileDrawer() {

    const currentModal = useModalStore((state) => state.currentModal);
    const modalData = useModalStore((state) => state.modalData);
    const closeModal = useModalStore((state) => state.closeModal);

    const ModalContent = currentModal ? modalComponents[currentModal].content : null;
    const height = currentModal ? modalComponents[currentModal].height : null
    const controls = useDragControls();

    const closeRef = useRef<{ handleClose: () => void }>(null);
    const handleClose = () => {
        closeRef.current?.handleClose();
        closeModal();
    }

    return (
        <LayoutGroup id={"layout-mobile"}>
            <AnimatePresence mode="wait" initial={false}>
                {currentModal && (
                    <div>
                        {/* Backdrop */}
                        <motion.div
                            key="mobile-drawer-root"
                            className="fixed inset-0 z-[1000] bg-neutral-950/70"
                            initial={{ opacity: 0 }}
                            animate={{ opacity: 1 }}
                            exit={{ opacity: 0 }}
                            onClick={handleClose}
                        />

                        {/* Sheet */}
                        <motion.div
                            className={`fixed bottom-0 left-0 right-0 z-[10000] w-full ${height} max-h-[calc(100%-5rem)] bg-gray-900 flex flex-col overflow-hidden`}
                            initial={{ y: "100%" }}
                            animate={{ y: 0 }}
                            exit={{ y: "100%" }}
                            // transition={{ type: "spring", stiffness: 600, damping: 40 }}
                            transition={{ duration: 0.22, ease: "easeOut" }}
                            drag="y"
                            dragConstraints={{ top: 0, bottom: 0 }}
                            dragElastic={{ top: 0, bottom: 1 }}
                            dragControls={controls}
                            dragListener={false}
                            onDragEnd={(_, info) => {
                                const shouldClose = info.offset.y > 90 || info.velocity.y > 200;
                                if (shouldClose) handleClose();
                            }}
                            onClick={(e) => e.stopPropagation()}
                        >
                            {/* Drag handle */}
                            <div
                                onPointerDown={(e) => {
                                    e.preventDefault();
                                    controls.start(e);
                                }}
                                style={{ touchAction: "none" }}
                                className="h-6 w-full flex items-center justify-center cursor-grab active:cursor-grabbing py-5"
                            >
                                <span className="w-9 h-[0.3rem] rounded-full bg-gray-600" />
                            </div>

                            {/* Scrollable content */}
                            <div className="min-h-0 overflow-y-auto overscroll-contain">
                                {ModalContent && <ModalContent {...modalData} ref={closeRef} />}
                            </div>
                        </motion.div>
                    </div>
                )}
            </AnimatePresence>
        </LayoutGroup>

    );
}
