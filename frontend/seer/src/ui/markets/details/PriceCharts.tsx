import { Outcome, Timeframe } from "@/lib/definitions";
import { usePrefs } from "@/lib/stores/prefs";
import { AnimatedOdds } from "@/ui/odds/AnimatedOdds";
import NumberFlow from "@number-flow/react";
import Decimal from "decimal.js";
import { AreaData, AreaSeries, createChart, CrosshairMode, LineStyle, PriceScaleMode, Time } from "lightweight-charts";
import { motion } from "motion/react";
import { useEffect, useRef, useState } from "react";

const colorsOutcomes = [
    { text: "text-blue-400", bg: "bg-blue-400", hex: "#3B82F6" },
    { text: "text-yellow-400", bg: "bg-yellow-400", hex: "#F59E0B" },

    { text: "text-purple-400", bg: "bg-purple-400", hex: "#A855F7" },
    { text: "text-green-400", bg: "bg-green-400", hex: "#10B981" },
    { text: "text-red-400", bg: "bg-red-400", hex: "#EF4444" },

];

const timeframes: Timeframe[] = ["24h", "7d", "30d", "all"];

export default function PriceCharts({ outcomes }: { outcomes: Outcome[] }) {

    const oddsFormat = usePrefs(state => state.oddsFormat);

    const chartContainerRef = useRef<HTMLDivElement>(null);
    const tooltipRef = useRef<HTMLDivElement>(null);

    // SELECT CSS VARIABLE --color-gray-400
    const textColor = "#99a1af"
    const [selectedPriceOutcome, setSelectedPriceOutcome] = useState<Record<string, Decimal> | null>(null);
    const [selectedDateFormatted, setSelectedDateFormatted] = useState<string>("");

    const seriesRefs = useRef<Record<string, AreaSeries>>({});

    const [selectedTimeframe, setSelectedTrimeframe] = useState<Timeframe>("24h");

    useEffect(
        () => {
            if (!chartContainerRef.current) return;
            const container = chartContainerRef.current;

            const chart = createChart(container, {
                layout: {
                    background: { color: "transparent" },
                    textColor: textColor,
                    fontSize: 10,
                },
                width: container.clientWidth,  // Set initial width
                height: 250,
                grid: {
                    vertLines: { visible: true, style: LineStyle.Dotted, color: '#222222' },
                    horzLines: { visible: true, style: LineStyle.Dotted, color: '#222222' },
                },
                rightPriceScale: {
                    visible: true,
                    autoScale: true,
                    borderVisible: false,
                    mode: PriceScaleMode.Normal,

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
                    borderVisible: false,
                },
            });

            outcomes.forEach((o, index) => {

                if (!o.priceCharts || o.priceCharts.length === 0) return;

                const colors = colorsOutcomes[index % colorsOutcomes.length];

                const series = chart.addSeries(AreaSeries, {
                    lineColor: colors.hex,
                    priceLineVisible: false,
                    topColor: "transparent",
                    lineWidth: 2,
                    lineStyle: LineStyle.Solid,
                    lastValueVisible: false,
                });

                series.priceScale().applyOptions({
                    autoScale: true,
                    scaleMargins: {
                        top: 0.05,
                        bottom: 0.05,
                    },
                });

                const chartData = o.priceCharts?.find(c => c.timeframe === selectedTimeframe)?.prices.map(point => ({
                    time: Math.floor(new Date(point.date).getTime() / 1000) as Time,
                    value: point.price.toNumber(),
                }));

                if (!chartData) return

                series.setData(chartData);
                seriesRefs.current[o.id] = series;
            });


            chart.timeScale().applyOptions({
                borderVisible: false,
                timeVisible: true,
                tickMarkFormatter: (time: Time) => {
                    const date = new Date((time as number) * 1000);

                    if (selectedTimeframe === '24h') {
                        return date.toLocaleTimeString('en-US', {
                            hour: '2-digit',
                            minute: '2-digit',
                            hour12: false
                        });
                    }

                    return date.toLocaleDateString('en-US', {
                        month: 'short',
                        day: 'numeric'
                    });
                },
                rightOffset: 1,
            });

            chart.applyOptions({
                localization: {
                    priceFormatter: (price: number) => (price * 100).toFixed(1) + '%',
                }
            });

            chart.timeScale().fitContent();

            const tooltip = tooltipRef.current;

            chart.subscribeCrosshairMove((param) => {
                if (
                    !param.time ||
                    param.point === undefined ||
                    param.point.x < 0 ||
                    param.point.y < 0
                ) {
                    setSelectedPriceOutcome(null);
                    if (tooltip) tooltip.style.display = 'none';
                    return;
                }



                const prices: Record<string, Decimal> = {};
                outcomes.forEach((o) => {
                    const series = seriesRefs.current[o.id];
                    const data = param.seriesData.get(series) as AreaData | undefined;
                    if (data) {
                        prices[o.id] = new Decimal(data.value);
                    }
                });





                if (tooltip) {
                    const toolTipMargin = 15;
                    const toolTipWidth = tooltip.offsetWidth;
                    const toolTipHeight = tooltip.offsetHeight;
                    tooltip.style.display = 'block';

                    const date = new Date((param.time as number) * 1000);
                    const formattedDate = date.toLocaleDateString('en-US', { day: '2-digit', month: 'short' }) + ", " + date.toLocaleTimeString('en-US', { hour: '2-digit', minute: '2-digit', hour12: false })

                    setSelectedDateFormatted(formattedDate);

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
                }

                setSelectedPriceOutcome(prices);

            });

            // Manual ResizeObserver
            const handleResize = () => {
                const width = container.clientWidth;
                console.log("resizing chart", "width", width);

                if (width > 0) {
                    chart.applyOptions({ width });
                    chart.timeScale().fitContent();
                }
            };

            const resizeObserver = new ResizeObserver(handleResize);
            resizeObserver.observe(container);
            window.addEventListener('resize', handleResize);

            return () => {
                resizeObserver.disconnect();
                window.removeEventListener('resize', handleResize);
                chart.remove();
            };
        },
        [outcomes, selectedTimeframe, textColor]
    );

    return (
        <div className="w-full">
            {/* Display outcomes with different colors, and their respective prices */}
            <div className="flex gap-6.5 text-sm mb-4 md:mb-8 flex-wrap">
                {outcomes.map((outcome, index) => (
                    <div key={outcome.id} className={`flex gap-2 items-center shrink-0 text-[13px]`}>
                        <span className={`rounded-full h-2 w-2 ${colorsOutcomes[index % colorsOutcomes.length].bg}`}></span>
                        <span className="font-normal text-gray-400">{outcome.name}</span>
                        <span className={`font-medium ${colorsOutcomes[index % colorsOutcomes.length].text} mb-[1px]`}>
                            <AnimatedOdds prob={selectedPriceOutcome?.[outcome.id] || outcome.priceYesNormalized} format={"percent"} />
                        </span>
                    </div>
                ))}
            </div>

            <div className="relative">
                <div ref={chartContainerRef} className="w-full" />
                <div
                    ref={tooltipRef}
                    className="bg-gray-700/70 absolute hidden rounded-lg shadow-lg p-3 z-10 backdrop-blur-sm"

                >
                    <p className="text-xs mb-1 text-gray-200">
                        {selectedDateFormatted}
                    </p>

                    <div className="flex flex-col">
                        {outcomes.map((outcome, index) => (
                            <div key={outcome.id} className={`flex gap-2 items-center justify-between text-sm`}>
                                <div className="flex items-center gap-1.5">
                                    <div className={`rounded-full h-2 w-2 mt-0.5 shrink-0 ${colorsOutcomes[index % colorsOutcomes.length].bg}`}></div>
                                    <span className="font-medium line-clamp-1 mr-3">{outcome.name}</span>
                                </div>
                                <span className={`font-bold ${colorsOutcomes[index % colorsOutcomes.length].text} mb-[1px]`}>
                                    <NumberFlow
                                        value={(selectedPriceOutcome?.[outcome.id] || outcome.priceYes).toNumber()}
                                        locales="en-US"
                                        format={{ style: 'percent', minimumFractionDigits: 1, maximumFractionDigits: 1, useGrouping: false }}
                                    />
                                </span>
                            </div>
                        ))}
                    </div>

                </div>
            </div>




            {/* Change timeframe */}
            <div className="flex gap-2 mt-5">
                {timeframes.map((t) => {
                    const isActive = selectedTimeframe === t;

                    return (
                        <button
                            key={t}
                            onClick={() => setSelectedTrimeframe(t)}
                            className={`${isActive ? "" : "hover:bg-gray-800 text-gray-300 cursor-pointer"
                                } relative w-12 py-1.5 rounded-md text-sm font-bold transition-colors delay-100`}
                        >
                            {/* Sliding Background */}
                            {isActive && (
                                <motion.div
                                    layoutId="active-pill"
                                    className="absolute inset-0 bg-primary-blue rounded-md"
                                    transition={{ ease: "easeInOut", duration: 0.2 }}
                                />
                            )}

                            {/* Text (Must be relative and z-10 to sit on top) */}
                            <span className="relative z-10 mix-blend-normal">
                                {t.toUpperCase()}
                            </span>
                        </button>
                    );
                })}
            </div>

        </div >
    )
}
