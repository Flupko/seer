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
                        `md:${height} w-10.5 h-10 justify-center md:justify-between md:pl-4 md:pr-4 md:w-full`
                        : `${height} justify-between pl-4 pr-4 w-full`}
                    ${bg} 
                    ${makeResponsive ? "md:hover:border-primary-blue" : "hover:border-primary-blue"} transition-all duration-200 rounded-full
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
                    <ChevronDown size={16} strokeWidth={2} className={`${makeResponsive ? "hidden md:block" : ""}`} />
                </motion.span>
            </button>

            {
                open && (
                    <motion.ul
                        initial={{ opacity: 0, scale: 0.96 }}
                        animate={{ opacity: 1, scale: 1 }}
                        exit={{ opacity: 0, scale: 0.96 }}
                        transition={{ duration: 0.15, ease: [0.4, 0, 0.2, 1] }}
                        className={`absolute top-[calc(100%+8px)] flex flex-col gap-1.5 z-20 bg-gray-800 rounded-xl p-2 overflow-hidden border border-gray-600 ${makeResponsive ? `${widthResponsive} ${positionResponsive} md:left-0 md:w-full` : "w-full left-0"}`}
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
                                        "w-full text-left px-3 py-2 transition-colors cursor-pointer text-sm rounded-lg relative flex items-center",
                                        active
                                            ? "bg-gray-700 text-primary-blue font-semibold"
                                            : "text-white hover:bg-gray-700 font-medium",
                                    ].join(" ")}
                                >
                                    {c.element}
                                    {active && <div className="absolute right-3 w-1.5 h-1.5 rounded-full bg-primary-blue"></div>}

                                </li>
                            );
                        })}
                    </motion.ul>
                )
            }
        </div >
    );
}
