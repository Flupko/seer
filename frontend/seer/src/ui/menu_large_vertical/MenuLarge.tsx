"use client"

import { HTMLMotionProps, motion } from "motion/react";
import MenuLargeItem from "./MenuLargeItem";
import { Settings, Trophy, Wallet } from "lucide-react";


export default function MenuLarge({ children, ...rest }: { children: React.ReactNode } & HTMLMotionProps<"div">) {
    return (
        <motion.div
            initial={{ opacity: 0}}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0}}
            transition={{ duration: 0.1, ease: 'easeInOut' }}
            className="w-55 bg-gray-900 rounded-lg border border-gray-600 py-3.5 px-4 overflow-y-scroll" {...rest}>
            {children}
        </motion.div>
    )
}



