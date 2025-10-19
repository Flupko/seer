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

    // Handle empty decimal separator cases
    if (out && (out[0] == "." || out[0] === ",")) return out;
    if (out.startsWith("0.") || out.startsWith("0,")) {
        return out; // keep "0." and "0," as is
    }

    // Remove unnecessary leading zeros
    if (out.startsWith("0") && out.length > 1) {
        // Remove all leading zeros
        const withoutLeadingZeros = out.replace(/^0+/, "");
        return withoutLeadingZeros || "0"; // if all were zeros, return "0"
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
}: PriceInputProps) {

    const [inner, setInner] = useState<string>(() => {
        const init = defaultValue != null ? String(defaultValue) : "";
        return sanitizeForEditing(init);
    });

    const setVal = (v: string) => {
        setInner(v);
    };


    const commitRound = () => {
        if (!inner) return;
        const hasComma = inner.includes(",");
        const normalized = hasComma ? inner.replace(",", ".") : inner;
        const roundedNorm = roundHalfDownStr(normalized, maxDecimals);
        setVal(roundedNorm);
        onValueChange?.(roundedNorm);
    }


    const onChange = (e: React.ChangeEvent<HTMLInputElement>) => {
        const next = sanitizeForEditing(e.target.value);
        setVal(next);
        if (next !== "") {
            onValueChange?.(next);
        } else {
            onValueChange?.(undefined);
        }
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
                className={`input-base w-full bg-gray-900 flex-nowrap rounded-md text-sm h-12 py-3 pl-10 font-bold
          border border-transparent outline-none focus:border-primary-blue placeholder:text-gray-400 placeholder:font-normal text-white`}
                {...rest}
            />
        </div>
    );
}
