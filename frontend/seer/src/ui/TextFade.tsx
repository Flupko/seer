"use client";
import { useEffect, useRef, useState } from "react";


export default function TextFade({ children, to = "to-gray-800" }: { children: React.ReactNode, to?: string }) {
    const [isOverflowing, setIsOverflowing] = useState(false);
    const spanRef = useRef<HTMLSpanElement>(null);

    useEffect(() => {
        if (spanRef.current) {
            const isOverflow = spanRef.current.scrollWidth > spanRef.current.clientWidth;
            setIsOverflowing(isOverflow);
        }

        const handleResize = () => {
            if (spanRef.current) {
                const isOverflow = spanRef.current.scrollWidth > spanRef.current.clientWidth;
                setIsOverflowing(isOverflow);
            }
        };

        window.addEventListener('resize', handleResize);
        return () => window.removeEventListener('resize', handleResize);
    }, [children]);

    return (
        <div className="flex overflow-hidden relative min-w-0">
            <span ref={spanRef} className="line-clamp-1 whitespace-nowrap">
                {children}
            </span>
            {isOverflowing && (
                <div
                    className={`absolute right-0 top-0 bottom-0 w-8 pointer-events-none z-10 bg-gradient-to-r from-transparent ${to}`}
                />
            )}
        </div>
    );

};