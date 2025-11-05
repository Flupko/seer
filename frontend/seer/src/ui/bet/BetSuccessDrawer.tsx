import { useDrawerStore } from "@/lib/stores/drawer";
import { motion } from "motion/react";
import Link from "next/link";
import Button from "../Button";
import DrawerHeader from "../drawer/DrawerHeader";


export default function BetSuccessDrawer(/* props */) {

    const closeDrawer = useDrawerStore((state) => state.closeDrawer);

    return (
        <>
            <DrawerHeader title="Success" />

            {/* Drawer body: column, take remaining height below header */}

            <div className="flex flex-col h-[calc(100vh-10rem)] items-center justify-center min-h-0 overflow-hidden px-5">
                {/* Replace 56px with your header height */}
                <div className="flex flex-col gap-4 items-center w-full">
                    {/* <Check className="w-32 h-32 filter-(--primary-blue-filter)" /> */}
                    <AnimatedCheckCircle />
                    <p className="text-gray-300 text-sm font-medium py-2 text-center">Your bet has been successfully placed.</p>
                    <div className="mt-3 w-full">

                        <Link className="contents" href="/mybets">
                            <Button bg="bg-neon-blue" width="full" onClick={closeDrawer}>
                                View Bets
                            </Button>
                        </Link>
                    </div>
                </div>
            </div>

        </>
    );
}

export function AnimatedCheckCircle() {
    const size = 100;
    const strokeWidth = 4;          // line thickness
    const radius = (size - strokeWidth) / 2;
    const circumference = 2 * Math.PI * radius;

    // circle dasharray: skip part of stroke for top-left gap
    const dashArray = circumference - 14;  // 14 ≈ small visual gap
    const dashOffset = circumference * 0.06; // offset for gap start

    return (
        <motion.svg
            width={size}
            height={size}
            viewBox={`0 0 ${size} ${size}`}
            className="text-primary-blue"
            initial="hidden"
            animate="visible"
            transition={{ staggerChildren: 0.1 }}
        >
            {/* Outer circle with a small gap (top-left) */}
            <motion.circle
                cx={size / 2}
                cy={size / 2}
                r={radius}
                fill="none"
                stroke="currentColor"
                strokeWidth={strokeWidth}
                strokeLinecap="round"
                strokeDasharray={`${dashArray} ${circumference - dashArray}`}
                strokeDashoffset={dashOffset}
                variants={{
                    hidden: { pathLength: 0, opacity: 0 },
                    visible: { pathLength: 1, opacity: 1 },
                }}
                transition={{
                    duration: 0.45,
                    ease: "easeInOut",
                }}
            />

            {/* Checkmark path */}
            <motion.path
                fill="none"
                stroke="currentColor"
                strokeWidth={strokeWidth}
                strokeLinecap="round"
                strokeLinejoin="round"
                d={`M${size * 0.32} ${size * 0.53} L${size * 0.45} ${size * 0.65} 
            L${size * 0.68} ${size * 0.38}`}
                variants={{
                    hidden: { pathLength: 0, opacity: 0 },
                    visible: { pathLength: 1, opacity: 1 },
                }}
                transition={{
                    duration: 0.3,
                    delay: 0.37,
                    ease: "easeOut",
                }}
            />
        </motion.svg>
    );
}