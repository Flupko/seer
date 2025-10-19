import { motion } from "motion/react";

export default function Tab({ isSelected, onClick, children }: { isSelected: boolean, onClick: () => void, children: React.ReactNode }) {
    return (
        <button className={`h-full text-sm flex-1 py-[0.625rem] flex align-center justify-center relative font-bold mb-2 cursor-pointer leading-1.5
         ${isSelected ? "text-primary-blue " : "text-white"}`} onClick={onClick}>
            {children}
            {isSelected ? (
                <motion.div
                    className="rounded-3xl absolute left-0 right-0 -bottom-[2.7px] h-[2.7px] bg-primary-blue"
                    layoutId="underline"
                    id="underline"


                    transition={{ duration: 0.2, ease: "easeOut" }}
                />
            ) : null}
        </button>
    )
}