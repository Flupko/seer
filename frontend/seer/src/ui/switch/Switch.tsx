"use client"

import { motion } from "motion/react";

export default function Switch({ isOn, toggle, disableToggle }: { isOn: boolean; toggle: () => void; disableToggle?: boolean }) {
    return (
        <button
            type="button"
            role="switch"
            aria-checked={isOn}
            onClick={() => !disableToggle && toggle()}
            className={`relative flex h-[24px] w-[42px] items-center rounded-[24px] transition-colors ${isOn ? "bg-neon-blue" : "bg-gray-600"
                } cursor-pointer shrink-0`}
        >
            <motion.span
                className="absolute left-[5px] h-[14px] w-[14px] rounded-full bg-white"
                animate={{ x: isOn ? 18 : 0 }}
                exit={{ x: isOn ? 18 : 0 }}
                transition={{ ease: "linear", duration: 0.15 }}
            />
        </button>
    )
}
