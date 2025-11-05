"use client";

import { OddsFormat } from "@/lib/odds";
import Decimal from "decimal.js";
import { motion, useAnimate } from "motion/react";
import { useEffect, useRef, useState } from "react";
import { AnimatedOdds } from "./AnimatedOdds";

export function ArrowOdds({
    prob,
    format,
}: {
    prob: Decimal;
    format: OddsFormat;
}) {
    const prevProb = useRef(prob);
    const [direction, setDirection] = useState<'up' | 'down' | null>(null);

    useEffect(() => {
        const cmp = prob.comparedTo(prevProb.current);
        if (cmp < 0) {
            setDirection('up');
        } else if (cmp > 0) {
            setDirection('down');
        }
        prevProb.current = prob;
    }, [prob]);

    return (
        <span className="flex items-center gap-1">
            <AnimatedOdds prob={prob} format={format} />
            {direction && (
                <OddDeltaArrow
                    direction={direction}
                    onComplete={() => setDirection(null)}
                />
            )}
        </span>
    );
}

function OddDeltaArrow({
    direction,
    onComplete,
}: {
    direction: 'up' | 'down';
    onComplete: () => void;
}) {
    const [scope, animate] = useAnimate();
    useEffect(() => {

        const controls = animate(
            scope.current,
            direction === 'up'
                ? { opacity: [0.1, 1, 0.1, 1, 0.1, 1, 0] }
                : { opacity: [0.1, 1, 0.1, 1, 0.1, 1, 0] },
            {
                duration: 2.6,
                ease: 'linear',
            }
        );

        controls.finished.then(() => {
            onComplete();
        });


    }, [scope, onComplete, direction]);

    return (
        <motion.span
            ref={scope}
            className={`w-2 h-2 flex items-center mt-[1px]`}
            aria-label={direction === 'up' ? 'odds up' : 'odds down'}
        >
            {direction === 'up' ? <CaretUpFilled /> : <CaretDownFilled />}
        </motion.span>
    );
}

export function CaretUpFilled({ color = "text-emerald-500" }: { color?: string }) {
    return (
        <svg
            clipRule="evenodd"
            fillRule="evenodd"
            strokeLinejoin="round"
            strokeMiterlimit="2"
            className={`${color} fill-current`}
            viewBox="6 7 12 9"
            xmlns="http://www.w3.org/2000/svg"
        >
            <path d="m16.843 13.789c.108.141.157.3.157.456 0 .389-.306.755-.749.755h-8.501c-.445 0-.75-.367-.75-.755 0-.157.05-.316.159-.457 1.203-1.554 3.252-4.199 4.258-5.498.142-.184.36-.29.592-.29.23 0 .449.107.591.291 1.002 1.299 3.044 3.945 4.243 5.498z" />
        </svg>
    );
}

export function CaretDownFilled({ color = "text-red-500" }: { color?: string }) {
    return (
        <svg
            clipRule="evenodd"
            fillRule="evenodd"
            strokeLinejoin="round"
            strokeMiterlimit="2"
            className={`${color} fill-current`}
            viewBox="6 8 12 9"
            xmlns="http://www.w3.org/2000/svg"
        >
            <path d="m16.843 10.211c.108-.141.157-.3.157-.456 0-.389-.306-.755-.749-.755h-8.501c-.445 0-.75.367-.75.755 0 .157.05.316.159.457 1.203 1.554 3.252 4.199 4.258 5.498.142.184.36.29.592.29.23 0 .449-.107.591-.291 1.002-1.299 3.044-3.945 4.243-5.498z" />
        </svg>
    );
}
