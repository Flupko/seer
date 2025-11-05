"use client";

import { DrawerType, useDrawerStore } from "@/lib/stores/drawer";
import { AnimatePresence, motion } from "motion/react";
import { useMediaQuery } from "usehooks-ts";
import BetDrawer from "../bet/BetDrawer";
import SuccessBetDrawer from "../bet/BetSuccessDrawer";
import ChatDrawer from "../chat/ChatDrawer";

export const drawerComponent: Record<Exclude<DrawerType, null>, React.ElementType> = {
    chat: ChatDrawer,
    bet: BetDrawer,
    betSuccess: SuccessBetDrawer,
};

export default function Drawer() {
    const currentDrawer = useDrawerStore((state) => state.currentDrawer);

    const DrawerContent = currentDrawer ? drawerComponent[currentDrawer] : null;
    const drawerData = useDrawerStore((state) => state.drawerData);

    const isSmall = useMediaQuery('(max-width: 1024px)');

    // Don't return early - let AnimatePresence handle the exit

    return (
        <>
            {/* Desktop Drawer */}
            <AnimatePresence mode="wait" initial={false}>
                {currentDrawer && !isSmall && (
                    <motion.aside
                        className="border-l border-l-gray-700 bg-gray-900 h-screen max-h-screen overflow-hidden"
                        key="desktop-drawer"
                        initial={{ width: 0 }}
                        animate={{ width: "22.5rem" }}
                        exit={{ width: 0 }}
                        transition={{ duration: 0.3, ease: "easeInOut" }}
                    >
                        <AnimatePresence mode="wait" initial={false}>
                            <motion.div className="w-90 h-screen max-h-screen"
                                key={currentDrawer}
                                initial={{ opacity: 0 }}
                                animate={{ opacity: 1 }}
                                exit={{ opacity: 0 }}
                                transition={{ duration: 0.07, ease: "linear" }}
                            >
                                {DrawerContent && <DrawerContent {...drawerData} />}
                            </motion.div>
                        </AnimatePresence>
                    </motion.aside>
                )}
            </AnimatePresence>

            {/* Mobile Drawer */}
            <AnimatePresence mode="wait" initial={false}>
                {currentDrawer && isSmall && (
                    <motion.aside
                        className="overflow-hidden z-50 h-[calc(100vh-60px)]"

                    >
                        <motion.div className="w-full fixed left-0 h-[calc(100vh-60px)] bg-gray-900"
                            key="mobile-drawer"
                            initial={{ y: "100%" }}
                            animate={{ y: 0 }}
                            exit={{ y: "100%" }}
                            transition={{ duration: 0.3, ease: "easeInOut" }}>
                            <div className="flex flex-col h-full relative">
                                {DrawerContent && <DrawerContent {...drawerData} />}
                            </div>

                        </motion.div>
                    </motion.aside>
                )}
            </AnimatePresence>
        </>
    );
}
