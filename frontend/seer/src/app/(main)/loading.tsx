"use client"
import Loader from "@/ui/loader/Loader"
import { useEffect, useState } from "react"

export default function Loading() {
    const [dots, setDots] = useState("")

    useEffect(() => {
        const id = setInterval(
            () => setDots(prev => (prev === "..." ? "" : prev + ".")),
            300
        )
        return () => clearInterval(id)
    }, [])

    return (
        <div className="h-[20vh] flex items-center justify-center">
            <div className="flex flex-col items-center gap-3.5">
                <Loader size={0.8} />
                <div className="text-gray-300 relative">
                    <span>Loading</span>
                    <span className="absolute left-full top-0 ml-1">{dots}</span>
                </div>
            </div>
        </div>
    )
}
