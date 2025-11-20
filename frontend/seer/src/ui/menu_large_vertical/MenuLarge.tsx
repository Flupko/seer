"use client"

import { HTMLMotionProps, motion } from "motion/react";


export default function MenuLarge({ children, ...rest }: { children: React.ReactNode } & HTMLMotionProps<"div">) {
    return (
        <motion.div
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            transition={{ duration: 0.12, ease: [0.4, 0, 0.2, 1] }}
            className="w-55 bg-grayscale-black rounded-xl border border-gray-700 py-3.5 px-4 overflow-y-scroll" {...rest}>
            {children}
        </motion.div>
    )
}



