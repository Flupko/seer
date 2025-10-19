"use client";

import { DrawerType, useDrawerStore } from "@/lib/stores/drawer";
import { AnimatePresence, motion } from "motion/react";
import { useMediaQuery } from "usehooks-ts";
import BetDrawer from "../bet/BetDrawer";

export const drawerComponent: Record<Exclude<DrawerType, null>, React.ElementType> = {
    chat: BetDrawer,
    bet: BetDrawer,
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
                        className="border-l border-l-gray-700 h-screen bg-gray-900 overflow-hidden flex-shrink-0"
                        key="desktop-drawer"
                        initial={{ width: 0 }}
                        animate={{ width: "22.5rem" }}
                        exit={{ width: 0 }}
                        transition={{ duration: 0.3, ease: "easeInOut" }}
                    >
                        <div className="w-90 h-full overflow-y-auto">
                            {DrawerContent && <DrawerContent {...drawerData} />}
                        </div>
                    </motion.aside>
                )}
            </AnimatePresence>

            {/* Mobile Drawer */}
            <AnimatePresence mode="wait" initial={false}>
                {currentDrawer && isSmall && (
                    <motion.aside
                        className="fixed inset-0 bg-gray-900 overflow-hidden z-[70]"
                        key="mobile-drawer"
                        initial={{ y: "100%" }}
                        animate={{ y: 0 }}
                        exit={{ y: "100%" }}
                        transition={{ duration: 0.3, ease: "easeInOut" }}
                    >
                        <div className="h-full overflow-y-auto">
                            {DrawerContent && <DrawerContent {...drawerData} />}
                        </div>
                    </motion.aside>
                )}
            </AnimatePresence>
        </>
    );
}
