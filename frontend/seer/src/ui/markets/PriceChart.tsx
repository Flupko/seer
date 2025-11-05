"use client"

import { PriceChartDataPoint } from "@/lib/definitions"
import Decimal from "decimal.js"
import { Area, AreaChart, ResponsiveContainer, Tooltip, TooltipProps } from "recharts"
import { AnimatedOdds } from "../odds/AnimatedOdds"

export const description = "A linear area chart"

export function PriceChart({ data }: { data: PriceChartDataPoint[] }) {
    return (
        <ResponsiveContainer width="100%" height={200} >
            <AreaChart
                data={data.map((point) => ({
                    date: point.date.toISOString(),
                    price: point.price.toNumber(),
                }))}

            >

                <Tooltip content={CustomTooltip} cursor={false} />
                <Area

                    dataKey="price"
                    type="linear"
                    fill="none"
                    stroke="var(--color-primary-blue)"
                />
            </AreaChart>
        </ResponsiveContainer>
    )
}

const CustomTooltip = ({ active, payload }: TooltipProps<number, string>) => {
    if (!active || !payload || !payload.length) return null

    const data = payload[0]
    const date = new Date(data.payload.date)

    return (
        <div className="bg-gray-700 rounded-lg shadow-lg p-2.5">
            <p className="text-xs text-gray-200">
                {/* Date format 04 Nov, 13:50 for example, in english BUT NOT AM / PM */}
                {date.toLocaleDateString('en-US', { day: '2-digit', month: 'short' })}, {date.toLocaleTimeString('en-US', { hour: '2-digit', minute: '2-digit', hour12: false })}

            </p>
            <p className="text-lg font-semibold text-primary-blue">
                <AnimatedOdds prob={new Decimal(data.value || 0)} format="percent" />
            </p>
        </div>
    )
}