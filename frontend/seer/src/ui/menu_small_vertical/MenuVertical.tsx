import { ChevronDown } from "lucide-react";
import { motion } from "motion/react";
import { useEffect, useRef, useState } from "react";

type MenuChoice = { value: any; element: React.ReactNode };

export default function MenuVertical({
    leftPart,
    choices,
    value,
    onChange,
    height = "h-12",
    bg = "bg-gray-800",
    makeResponsive = false,
    widthResponsive,
    positionResponsive,
}: {
    leftPart?: React.ReactNode;
    value: any;
    onChange: (v: any) => void;
    choices: MenuChoice[];
    height?: string;
    bg?: string;
    makeResponsive?: boolean;
    widthResponsive?: string;
    positionResponsive?: string;
}) {

    const [open, setOpen] = useState(false);
    const wrapRef = useRef<HTMLDivElement | null>(null);

    useEffect(() => {
        const onDown = (e: MouseEvent | PointerEvent) => {
            if (!wrapRef.current) return;
            if (!wrapRef.current.contains(e.target as Node)) setOpen(false);
        };
        const onKey = (e: KeyboardEvent) => {
            if (e.key === "Escape") setOpen(false);
        };
        document.addEventListener("mousedown", onDown);
        document.addEventListener("pointerdown", onDown);
        document.addEventListener("keydown", onKey);
        return () => {
            document.removeEventListener("mousedown", onDown);
            document.removeEventListener("pointerdown", onDown);
            document.removeEventListener("keydown", onKey);
        };
    }, []);



    return (
        <div ref={wrapRef} className={`relative ${makeResponsive ? "md:w-full" : "w-full"}`}>
            <button
                type="button"
                aria-haspopup="menu"
                aria-expanded={open}
                onClick={() => setOpen((p) => !p)}
                className={`cursor-pointer border 
                ${makeResponsive ?
                        `md:${height} w-10.5 h-10.5 justify-center md:justify-between md:px-4 md:w-full`
                        : `${height} justify-between px-4 w-full`}
                    ${bg} 
                    ${makeResponsive ? "md:hover:border-primary-blue" : "hover:border-primary-blue"} transition-all duration-200 rounded-md 
                    items-center text-left flex 
                    ${open ? `${makeResponsive ? "md:border-primary-blue border-transparent" : "border-primary-blue"}` : "border-transparent"}`}
            >
                <div className="text-sm font-medium flex gap-3 items-center">
                    {leftPart}
                    <span className={`${makeResponsive ? "hidden md:block" : ""}`}>
                        {choices.find((c) => c.value === value)?.element}
                    </span>

                </div>
                <motion.span
                    style={{ display: "inline-flex" }}
                    animate={{ rotate: open ? 180 : 0 }}
                    transition={{ duration: 0.13, ease: "linear" }}
                >
                    <ChevronDown size={22.5} strokeWidth={1.2} className={`${makeResponsive ? "hidden md:block" : ""}`} />
                </motion.span>
            </button>

            {
                open && (
                    <motion.ul
                        initial={{ opacity: 0, scale: 0.96 }}
                        animate={{ opacity: 1, scale: 1 }}
                        exit={{ opacity: 0, scale: 0.96 }}
                        transition={{ duration: 0.15, ease: [0.4, 0, 0.2, 1] }}
                        className={`absolute top-14 z-20 bg-gray-800 rounded-md overflow-hidden border border-gray-600 ${makeResponsive ? `${widthResponsive} ${positionResponsive} md:left-0 md:w-full` : "w-full left-0"}`}
                    >
                        {choices.map((c) => {
                            const active = c.value === value;
                            return (
                                <li
                                    key={c.value}
                                    onClick={() => {
                                        onChange(c.value);
                                        setOpen(false);
                                    }}
                                    className={[
                                        "w-full text-left px-6 py-3 transition-colors border-b border-gray-600 last:border-none cursor-pointer text-sm",
                                        active
                                            ? "bg-gray-700 text-primary-blue font-bold"
                                            : "text-white hover:bg-gray-700 font-medium",
                                    ].join(" ")}
                                >
                                    {c.element}
                                </li>
                            );
                        })}
                    </motion.ul>
                )
            }
        </div >
    );
}
