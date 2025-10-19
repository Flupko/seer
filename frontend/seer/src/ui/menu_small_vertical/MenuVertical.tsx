import { ChevronDown } from "lucide-react";
import { motion } from "motion/react";
import { useEffect, useRef, useState } from "react";

type MenuChoice = { value: string; element: React.ReactNode };

export default function MenuVertical({
    leftPart,
    choices,
    value,
    onChange,
    height
}: {
    leftPart?: React.ReactNode;
    value: string;
    onChange: (v: string) => void;
    choices: MenuChoice[];
    height?: string
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
        <div ref={wrapRef} className="relative w-full">
            <button
                type="button"
                aria-haspopup="menu"
                aria-expanded={open}
                onClick={() => setOpen((p) => !p)}
                className={`cursor-pointer border ${height ? height : "h-12"} px-4 bg-gray-800 hover:border-primary-blue transition-all duration-200 rounded-md w-full items-center text-left flex justify-between
                    ${open ? "border-primary-blue" : "border-transparent"}`}
            >
                <div className="text-sm font-medium flex gap-3 items-center">
                    {leftPart}
                    {choices.find((c) => c.value === value)?.element}
                </div>
                <motion.span
                    style={{ display: "inline-flex" }}
                    animate={{ rotate: open ? 180 : 0 }}
                    transition={{ duration: 0.15, ease: "easeInOut" }}
                >
                    <ChevronDown size={22.5} strokeWidth={1.2} />
                </motion.span>
            </button>

            {open && (
                <motion.ul
                    initial={{ opacity: 0, scale: 0.96 }}
                    animate={{ opacity: 1, scale: 1 }}
                    exit={{ opacity: 0, scale: 0.96 }}
                    transition={{ duration: 0.15, ease: [0.4, 0, 0.2, 1] }}
                    className="absolute top-14 right-0 z-10 w-full bg-gray-800 rounded-md overflow-hidden border border-gray-600"
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
            )}
        </div>
    );
}
