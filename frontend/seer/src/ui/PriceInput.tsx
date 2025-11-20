"use client";

import Image from "next/image";
import React, { useState } from "react";

type PriceInputProps = {
    maxDecimals?: number;
    defaultValue?: string | number;
    onValueChange?: (value: string | undefined) => void; // emits sanitized/rounded string
} & React.InputHTMLAttributes<HTMLInputElement>;

function sanitizeForEditing(raw: string) {

    // Keep digits and a single separator (dot or comma)
    let out = "";
    let seenSep = false;

    for (const ch of raw) {
        if (ch >= "0" && ch <= "9") out += ch;
        else if ((ch === "." || ch === ",") && !seenSep) {
            out += ch; // preserve user's separator
            seenSep = true;
        }
    }

    return out;
}


// increment an integer string by 1
function incIntString(s: string): string {
    if (!s) return "1";
    const arr = s.split("");
    let carry = 1;
    for (let i = arr.length - 1; i >= 0 && carry; i--) {
        const d = parseInt(arr[i]) + carry;
        if (d >= 10) {
            arr[i] = "0";
            carry = 1;
        } else {
            arr[i] = String.fromCharCode(48 + d);
            carry = 0;
        }
    }
    if (carry) arr.unshift("1");
    return arr.join("");
}

function addOne(intStr: string, fracStr: string) {
    // increment (intStr.fracStr) by 1 in the fractional precision
    let carry = 1;
    const arr = fracStr.split("");
    for (let i = arr.length - 1; i >= 0 && carry; i--) {
        const d = parseInt(arr[i]) + carry;
        if (d >= 10) {
            arr[i] = "0";
            carry = 1;
        } else {
            arr[i] = String.fromCharCode(48 + d);
            carry = 0;
        }
    }
    let newInt = intStr;
    if (carry) newInt = incIntString(intStr);
    return { intStr: newInt, fracStr: arr.join("") };
}

function toFixedDp(intPart: string, fracPart: string, dp: number): string {
    const padded = (fracPart || "").padEnd(dp, "0").slice(0, dp);
    return `${intPart}.${padded}`; // normalized uses "."
}

// Round-half-down for positive numbers on a normalized dot string
function roundHalfDownStr(normalizedDot: string, dp: number): string {
    if (!normalizedDot) return "";
    console.log("roundHalfDownStr", normalizedDot, dp);
    const parts = normalizedDot.split(".");
    const intPart = parts[0] || "0";
    const fracPart = parts[1] || "";

    if (fracPart.length <= dp) {
        return toFixedDp(intPart, fracPart, dp);
    }

    const keep = fracPart.slice(0, dp);
    const rest = fracPart.slice(dp);

    const isTie = rest[0] === "5" && /^[0]*$/.test(rest.slice(1));
    if (isTie) return toFixedDp(intPart, keep, dp);

    const shouldUp = rest[0] > "5" || (rest[0] === "5" && /[1-9]/.test(rest.slice(1)));
    if (!shouldUp) return toFixedDp(intPart, keep, dp);

    const { intStr, fracStr } = addOne(intPart, keep);
    const roundedFrac = (fracStr || "").padStart(dp, "0").slice(0, dp);
    return dp > 0 ? `${intStr}.${roundedFrac}` : intStr;
}

export default function PriceInput({
    maxDecimals = 2,
    defaultValue,
    onValueChange,
    ...rest
}: PriceInputProps & { ref: React.Ref<HTMLInputElement> | null }) {

    const [inner, setInner] = useState<string>(() => {
        const init = defaultValue != null ? String(defaultValue) : "";
        return sanitizeForEditing(init);
    });



    const commitRound = () => {
        if (!inner) return;

        const normalized = inner.replace(",", ".");

        // Remove unnecessary leading zeros
        const parts = normalized.split(".");
        let intPart = parts[0];
        const fracPart = parts[1] || "";

        // Remove leading zeros in integer part
        intPart = intPart.replace(/^0+/, "");
        if (intPart === "") intPart = "0";

        const normalizedNoLeading = fracPart ? `${intPart}.${fracPart}` : intPart;

        const roundedNorm = roundHalfDownStr(normalizedNoLeading, maxDecimals);
        setInner(roundedNorm);
    }


    const onChange = (e: React.ChangeEvent<HTMLInputElement>) => {
        const next = sanitizeForEditing(e.target.value);
        setInner(next);
        if (!next) {
            onValueChange?.(undefined);
            return;
        }

        const normalized = next.replace(",", ".");
        const roundedNorm = roundHalfDownStr(normalized, maxDecimals);
        onValueChange?.(roundedNorm);

    }

    return (
        <div className="flex justify-between items-start relative w-full gap-1">
            <span className="flex justify-center items-center absolute h-full left-4">
                <Image src={"/icons/dollar.svg"} alt="Dollar" width={16} height={16} />
            </span>
            <input
                type="text"
                inputMode="decimal"
                pattern="^[0-9]*[.,]?[0-9]*$"
                value={inner}
                onChange={onChange}
                onBlur={commitRound}
                className={`input-base w-full bg-gray-800 flex-nowrap rounded-lg text-sm h-12 py-3 pl-10 font-bold
          border border-transparent outline-none focus:border-primary-blue placeholder:text-gray-500 placeholder:font-normal text-white
          transition-colors duration-100`}
                {...rest}
            />
        </div>
    );
}
