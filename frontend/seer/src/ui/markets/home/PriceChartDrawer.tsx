"use client"

import { BetSide, PriceChartDataPoint } from "@/lib/definitions";
import { AnimatedOdds } from "@/ui/odds/AnimatedOdds";
import Decimal from "decimal.js";
import { AreaData, AreaSeries, createChart, CrosshairMode, DeepPartial, LineStyle, LineWidth, Time } from 'lightweight-charts';
import { useEffect, useMemo, useRef, useState } from "react";

export function PriceChartDrawer({ data, side }: { data: PriceChartDataPoint[], side: BetSide }) {

    const chartContainerRef = useRef<HTMLDivElement>(null);
    const tooltipRef = useRef<HTMLDivElement>(null);

    const [selectedDateFormatted, setSelectedDateFormatted] = useState<string>("");
    const [priceTooltip, setPriceTooltip] = useState<Decimal>(new Decimal(0));

    const backgroundColor = getComputedStyle(document.documentElement).getPropertyValue('--color-gray-900').trim();
    const lineColor = getComputedStyle(document.documentElement).getPropertyValue('--color-primary-blue').trim();
    const primaryBlue = getComputedStyle(document.documentElement).getPropertyValue('--color-primary-blue').trim()
    const noenBlue = getComputedStyle(document.documentElement).getPropertyValue('--color-neon-blue').trim()
    const areaBottomColor = getComputedStyle(document.documentElement).getPropertyValue('--color-neon-blue').trim()

    const chartData = useMemo(() =>
        data.map((point) => ({
            time: Math.floor(new Date(point.date).getTime() / 1000) as Time,
            value: side === "y" ? point.price.toNumber() : 1 - point.price.toNumber(),
        })),
        [data, side]
    );

    useEffect(
        () => {
            if (!chartContainerRef.current) return;
            const container = chartContainerRef.current;

            const chart = createChart(container, {
                layout: {
                    background: { color: "transparent" },
                    textColor: "transparent",
                },
                width: container.clientWidth,
                height: 100,
                grid: {
                    vertLines: { visible: false },
                    horzLines: { visible: false },
                },
                rightPriceScale: {
                    visible: false,
                    autoScale: false,
                    // Add autoScale to fit data
                },
                timeScale: {
                    visible: false,
                    borderVisible: false,
                },
                crosshair: {
                    vertLine: {
                        visible: false,
                        labelVisible: false,
                    },
                    horzLine: {
                        visible: false,
                        labelVisible: false,
                    },
                    mode: CrosshairMode.Normal,
                },
                handleScroll: false,
                handleScale: false,
                overlayPriceScales: {
                    borderVisible: false
                },
            });

            chart.timeScale().fitContent();

            const series = chart.addSeries(AreaSeries, {
                lineColor: noenBlue,
                priceLineVisible: false,
                topColor: primaryBlue + '2D',
                bottomColor: primaryBlue + '00',
                lineWidth: 2 as DeepPartial<LineWidth>,
                lineStyle: LineStyle.Solid,
            });

            series.priceScale().applyOptions({
                autoScale: true,  // Automatically fit to your 0-1 data
                scaleMargins: {
                    top: 0.1,
                    bottom: 0.1,
                },
            });

            series.setData(chartData);
            chart.timeScale().fitContent();

            const tooltip = tooltipRef.current;
            if (tooltip) {

                chart.subscribeCrosshairMove((param) => {
                    if (
                        !param.time ||
                        param.point === undefined ||
                        param.point.x < 0 ||
                        param.point.y < 0
                    ) {
                        tooltip.style.display = 'none';
                        return;
                    }

                    const data = param.seriesData.get(series) as AreaData | undefined;
                    if (!data) {
                        tooltip.style.display = 'none';
                        return;
                    }

                    const toolTipMargin = 15;
                    const toolTipWidth = tooltip.offsetWidth;
                    const toolTipHeight = tooltip.offsetHeight;

                    tooltip.style.display = 'block';

                    const date = new Date((data.time as number) * 1000);

                    const formattedDate = date.toLocaleDateString('en-US', { day: '2-digit', month: 'short' }) + ", " + date.toLocaleTimeString('en-US', { hour: '2-digit', minute: '2-digit', hour12: false })
                    setSelectedDateFormatted(formattedDate);

                    setPriceTooltip(new Decimal(data.value));
                    const y = param.point.y;
                    let left = param.point.x + toolTipMargin;
                    if (left > container.clientWidth - toolTipWidth) {
                        left = param.point.x - toolTipMargin - toolTipWidth;
                    }


                    let top = y + toolTipMargin;
                    if (top > container.clientHeight - toolTipHeight) {
                        top = y - toolTipHeight - toolTipMargin;
                    }
                    tooltip.style.left = left + 'px';
                    tooltip.style.top = top + 'px';
                });
            }

            const handleResize = () => {
                chart.applyOptions({ width: container.clientWidth });
            };

            window.addEventListener('resize', handleResize);

            return () => {
                window.removeEventListener('resize', handleResize);
                chart.remove();
            };
        },
        [data, backgroundColor, lineColor, areaBottomColor, chartData, side]
    );

    return (
        <div className="relative">
            <div ref={chartContainerRef} />
            <div
                ref={tooltipRef}
                className="bg-gray-700/70 absolute hidden rounded-lg shadow-lg p-2.5 z-10 backdrop-blur-sm"

            >
                <p className="text-xs text-gray-200">
                    {selectedDateFormatted}
                </p>
                <p className="text-lg font-bold text-primary-blue">
                    <AnimatedOdds prob={priceTooltip} format="percent" />
                </p>
            </div>
        </div>
    );
}

// "use client"

// import { PriceChartDataPoint } from "@/lib/definitions"
// import Decimal from "decimal.js"
// import { Area, AreaChart, ResponsiveContainer, Tooltip, TooltipProps, YAxis } from "recharts"
// import { AnimatedOdds } from "../../odds/AnimatedOdds"

// export const description = "A linear area chart"

// export function PriceChartDrawer({ data }: { data: PriceChartDataPoint[] }) {
//     const chartData = data.map((point) => ({
//         date: point.date.toISOString(),
//         price: point.price.toNumber(),
//     }))

//     return (
//         <ResponsiveContainer width="100%" height={100}>
//             <AreaChart data={chartData}>
//                 {/* Define the gradient once and reference it via url(#priceFill) */}
//                 <defs>
//                     <linearGradient id="priceFill" x1="0" y1={Math.min(...chartData.map((d) => d.price))} x2="0" y2={Math.max(...chartData.map((d) => d.price))}>
//                         <stop offset="0%" stopColor="var(--color-neon-blue)" stopOpacity={0.2} />
//                         <stop offset="100%" stopColor="var(--color-neon-blue)" stopOpacity={0} />
//                     </linearGradient>
//                 </defs>

//                 <Tooltip content={CustomTooltip} cursor={false} />
//                 <YAxis type="number" domain={['dataMin', 'dataMax']} hide />

//                 <Area
//                     dataKey="price"

//                     stroke="var(--color-neon-blue)"
//                     strokeWidth={1.2}
//                     fill="url(#priceFill)"
//                     // Uncomment if you want the area to fill to chart bottom rather than y=0:
//                     // baseValue="dataMin"
//                     isAnimationActive={false}
//                 />
//             </AreaChart>
//         </ResponsiveContainer>
//     )
// }

// const CustomTooltip = ({ active, payload }: TooltipProps<number, string>) => {
//     if (!active || !payload || !payload.length) return null
//     const data = payload[0]
//     const date = new Date(data.payload.date)
//     return (
//         <div className="bg-gray-700 rounded-lg shadow-lg p-2.5">
//             <p className="text-xs text-gray-200">
//                 {date.toLocaleDateString('en-US', { day: '2-digit', month: 'short' })},{" "}
//                 {date.toLocaleTimeString('en-US', { hour: '2-digit', minute: '2-digit', hour12: false })}
//             </p>
//             <p className="text-lg font-semibold text-primary-blue">
//                 <AnimatedOdds prob={new Decimal(data.value || 0)} format="percent" />
//             </p>
//         </div>
//     )
// }
