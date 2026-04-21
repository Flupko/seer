import { ChevronDown } from "lucide-react";
import { AnimatePresence, motion } from "motion/react";
import { useEffect, useRef, useState } from "react";
import { createPortal } from "react-dom";

type MenuChoice = { value: any; element: React.ReactNode };

export default function MenuVertical({
    leftPart,
    choices,
    value,
    onChange,
    height = "h-12",
    padding = "px-4",
    bg = "bg-gray-800",
    menuWidth, // New optional prop (number in pixels)
}: {
    leftPart?: React.ReactNode;
    value: any;
    onChange: (v: any) => void;
    choices: MenuChoice[];
    height?: string;
    padding?: string;
    bg?: string;
    menuWidth?: number;
}) {
    const [open, setOpen] = useState(false);
    const buttonRef = useRef<HTMLButtonElement | null>(null);
    const [coords, setCoords] = useState({ top: 0, left: 0, width: 0 });

    const toggleMenu = () => {
        if (open) {
            setOpen(false);
            return;
        }

        if (buttonRef.current) {
            const rect = buttonRef.current.getBoundingClientRect();
            const PADDING = 12; // Minimum distance from screen edge

            // 1. Determine desired width (prop or match button)
            const finalWidth = menuWidth || rect.width;

            // 2. Initial Position: Align Left
            let leftPos = rect.left;

            // 3. Check Right Overflow
            // If (Left Edge + Width) > Screen Width, push it left
            if (leftPos + finalWidth > window.innerWidth - PADDING) {
                leftPos = window.innerWidth - finalWidth - PADDING;
            }

            // 4. Check Left Overflow (Safety for small mobile screens)
            // If pushing it left made it go off-screen to the left, reset to padding
            if (leftPos < PADDING) {
                leftPos = PADDING;
            }

            setCoords({
                top: rect.bottom + 8, // 8px vertical gap
                left: leftPos,
                width: finalWidth
            });

            setOpen(true);
        }
    };

    // Close on click outside
    useEffect(() => {
        const onDown = (e: MouseEvent | PointerEvent) => {
            if (buttonRef.current && buttonRef.current.contains(e.target as Node)) {
                return;
            }
            setOpen(false);
        };
        if (open) {
            document.addEventListener("mousedown", onDown);
            document.addEventListener("pointerdown", onDown);
        }
        return () => {
            document.removeEventListener("mousedown", onDown);
            document.removeEventListener("pointerdown", onDown);
        };
    }, [open]);

    // Close on scroll/resize
    useEffect(() => {
        const close = () => setOpen(false);
        window.addEventListener("scroll", close);
        window.addEventListener("resize", close);
        return () => {
            window.removeEventListener("scroll", close);
            window.removeEventListener("resize", close);
        }
    }, [])

    return (
        <>
            <button
                ref={buttonRef}
                type="button"
                onClick={toggleMenu}
                className={`cursor-pointer border 
                     ${height} justify-between ${padding} w-full
                    ${bg} 
                 hover:border-primary-blue transition-all duration-200 rounded-full
            items-center text-left flex
            ${open ? "border-primary-blue" : "border-transparent"}`}
            >
                <div className="text-sm font-medium flex items-center">
                    {leftPart}
                    <span>{choices.find((c) => c.value === value)?.element}</span>
                </div>
                <motion.span
                    className="ml-2"
                    style={{ display: "inline-flex" }}
                    animate={{ rotate: open ? 180 : 0 }}
                    transition={{ duration: 0.13, ease: "linear" }}
                >
                    <ChevronDown size={16} strokeWidth={2} />
                </motion.span>
            </button>

            {createPortal(
                <AnimatePresence>
                    {open && (
                        <motion.ul
                            initial={{ opacity: 0, scale: 0.96, originY: 0 }} // added originY top
                            animate={{ opacity: 1, scale: 1 }}
                            exit={{ opacity: 0, scale: 0.96 }}
                            transition={{ duration: 0.15, ease: [0.4, 0, 0.2, 1] }}
                            style={{
                                position: "fixed",
                                top: coords.top,
                                left: coords.left,
                                width: coords.width, // Applied here
                            }}
                            className={`flex flex-col gap-1.5 z-50 bg-gray-800 rounded-xl p-2 overflow-hidden border border-gray-600 shadow-xl`}
                        >
                            {choices.map((c) => {
                                const active = c.value === value;
                                return (
                                    <li
                                        key={c.value}
                                        onClick={(e) => {
                                            e.stopPropagation();
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
                                        {active && (
                                            <div className="absolute right-3 w-1.5 h-1.5 rounded-full bg-primary-blue"></div>
                                        )}
                                    </li>
                                );
                            })}
                        </motion.ul>
                    )}
                </AnimatePresence>,
                document.body
            )}
        </>
    );
}