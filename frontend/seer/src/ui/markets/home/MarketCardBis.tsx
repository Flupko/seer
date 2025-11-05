import { useWebSocket } from "@/app/WsProvider";
import { getMarket } from "@/lib/api";
import { MarketView } from "@/lib/definitions"; // Import from your types file
import { useOdds } from "@/lib/hooks/useOdds";
import { useDrawerStore } from "@/lib/stores/drawer";
import { formatDateTime } from "@/lib/utils/date";
import { ArrowOdds } from "@/ui/odds/ArrowOdds";
import NumberFlow from "@number-flow/react";
import { useQuery } from "@tanstack/react-query";
import { Bookmark, ChartColumnIncreasing, CheckCircle2, CircleX, Clock } from "lucide-react";
import Image from "next/image";
import { useEffect, useState } from "react";

export default function MarketCardBis({ marketInitial }: { marketInitial: MarketView }) {


    const [showAllOutcomes, setShowAllOutcomes] = useState(false);
    const { oddsFormat } = useOdds();

    const openDrawer = useDrawerStore(state => state.openDrawer);
    const closeDrawer = useDrawerStore(state => state.closeDrawer);

    const { data: market } = useQuery({
        queryKey: ['market', marketInitial.id],
        queryFn: () => getMarket(marketInitial.id),
        initialData: marketInitial,
        staleTime: Infinity,
    })


    const ws = useWebSocket();

    // On mount add the listener
    useEffect(() => {
        if (!ws) return;

        ws.emit({ type: "get:market_state", payload: { marketID: market.id } });

        const off = ws.onConnect(() => {
            ws.emit({ type: "get:market_state", payload: { marketID: market.id } });
        });

        return () => {
            off();
        }

    }, [ws, market.id]);


    // Sort outcomes based on outcomeSort preference
    const sortedOutcomes = [...market.outcomes].sort((a, b) => {
        if (market.outcomeSort === 'price') {
            return b.price.minus(a.price).toNumber();
        }
        return a.position - b.position;
    });

    // Top 3 outcomes for main display
    const topOutcomes = sortedOutcomes.slice(0, 3);
    const remainingOutcomes = sortedOutcomes.slice(3);

    // Convert probPPM to formatted odds
    const timeInfo = market.closeTime ? formatDateTime(market.closeTime) : null;

    return (
        <div
            className="bg-gray-800 rounded-lg overflow-hidden py-4 hover:bg-gray-700/80 transition-all duration-200"
        >
            {/* Header */}
            <div className="px-4 mb-6">
                <div className="flex gap-4">
                    {/* Compact Thumbnail */}
                    {market.imgKey && (
                        <div className="flex-shrink-0 w-11 h-11 rounded-md overflow-hidden bg-gray-900">
                            <Image
                                src={market.imgKey}
                                alt={market.name}
                                width={48}
                                height={48}
                                className="object-cover w-full h-full"
                            />
                        </div>
                    )}

                    {/* Market Info */}
                    <div className="flex-1 min-w-0">

                        <h3 className="text-sm font-bold text-white leading-tight line-clamp-2 transition-colors hover:underline hover:cursor-pointer mb-2.5">
                            {market.name}
                        </h3>

                        {/* Metadata Row */}
                        <div className="flex items-center gap-2.5 text-gray-400 font-medium">

                            <div className="px-2 py-0.5 bg-primary-blue/20 rounded-sm text-xs text-primary-blue flex ites-center">
                                {market.categories[0]?.label}
                            </div>

                            {timeInfo && (
                                <div className={`flex items-center text-xs font-normal gap-1 text-gray-400`}>
                                    <Clock size={11} strokeWidth={2} />
                                    <span className="line-clamp-1">{timeInfo}</span>
                                </div>
                            )}

                        </div>


                    </div>
                </div>
            </div>

            {/* Odds Section */}
            <div className="px-4 space-y-1.5 mb-5">
                <div className="flex gap-2.5">
                    {topOutcomes.map((outcome, idx) => {

                        // const bg = idx === 0 ? "bg-red-400/20" : "bg-green-400/20";
                        // const bgHover = idx === 0 ? "hover:bg-red-400/90" : "hover:bg-green-400/90";
                        // const outcomeColor = idx === 0 ? "text-red-500" : "text-green-500";

                        return (
                            <button
                                key={outcome.id}
                                disabled={market.status !== 'active'}
                                className={`flex-1 min-w-0 flex flex-col gap-1.5 items-start rounded-md 
                                    bg-grayscale-black
                                    ${market.status === 'active' && "cursor-pointer hover:bg-primary-blue active:bg-blue-pressed group active:scale-95"}
                                    ${market.status === "resolved" && market.resolution?.winningOutcomeId !== outcome.id && "brightness-40"}
                                    transition-all duration-200 px-3 py-2.5 `}
                                onClick={() => {
                                    if (market.status !== 'active') return;
                                    closeDrawer();
                                    // Wait some time
                                    openDrawer('bet', { marketId: market.id, initialOutcomeId: outcome.id });
                                }}
                            >
                                <div className="text-sm font-bold pr-3 text-start flex items-center gap-1.5">
                                    <span className="line-clamp-1">{outcome.name}</span>

                                    {market.status === "resolved" && (market.resolution?.winningOutcomeId === outcome.id ?
                                        <span><CheckCircle2 className="text-success" size={14} /></span> :
                                        <span><CircleX className="text-red-400" size={14} /></span>)
                                    }

                                </div>
                                <div className={`font-bold text-sm mr-2 text-primary-blue group-hover:text-white`}>
                                    <ArrowOdds
                                        prob={outcome.price}
                                        format={oddsFormat}
                                    />
                                </div>

                            </button>
                        );
                    })}
                </div>
            </div>


            <div className="flex justify-between items-center px-4 font-normal">

                <div className="flex items-center gap-2 text-xs">
                    {market.totalVolume && (
                        <div className="flex gap-1.5 text-success items-center">
                            <ChartColumnIncreasing strokeWidth={2} size={12} className="mt-0.5" />

                            <span className="flex gap-1 items-center">

                                <NumberFlow locales={"en-US"}
                                    className=""
                                    value={market.totalVolume.toNumber()}
                                    format={{ style: 'currency', currency: 'USD', minimumFractionDigits: 2, maximumFractionDigits: 2 }} />

                                <span>
                                    Vol.
                                </span>
                            </span>
                        </div>
                    )}


                </div>



                <span>
                    <Bookmark size={16} className="text-primary-blue" />
                </span>
            </div>


        </div>
    );
}
