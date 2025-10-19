import { useWebSocket } from "@/app/WsProvider";
import { getMarket } from "@/lib/api";
import { MarketView } from "@/lib/definitions"; // Import from your types file
import { useOdds } from "@/lib/hooks/useOdds";
import { useDrawerStore } from "@/lib/stores/drawer";
import { AnimatedOdds } from "@/ui/AnimatedOdds";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Clock, TrendingUp } from "lucide-react";
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
    const qc = useQueryClient();

    // On mount add the listener
    useEffect(() => {

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
    const timeInfo = market.closeTime;

    return (
        <div
            className="bg-gray-800 rounded-lg cursor-pointer overflow-hidden hover:bg-gray-700/80 transition-all duration-200"
        >
            {/* Header */}
            <div className="p-4 mb-3">
                <div className="flex gap-3 items-center">
                    {/* Compact Thumbnail */}
                    {market.imgKey && (
                        <div className="flex-shrink-0 w-12 h-12 rounded-md overflow-hidden bg-gray-900">
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
                        <h3 className="text-sm font-bold text-white leading-tight line-clamp-2 mb-2 transition-colors">
                            {market.name}
                        </h3>

                        {/* Metadata Row */}
                        <div className="flex items-center gap-2 text-gray-400 text-[11px] font-bold">
                            {market.categories.length > 0 && (
                                <div className="text-white flex items-center bg-[#22946e] font-black p-1.5 rounded-sm h-5">
                                    {market.categories[0].label.toUpperCase()}
                                </div>
                            )}
                            {timeInfo && (
                                <div className={`flex items-center p-1.5 gap-1 text-white rounded-sm bg-gray-500 h-5`}>
                                    <Clock size={11} strokeWidth={2.5} />
                                    <span>{timeInfo}</span>
                                </div>
                            )}

                            {sortedOutcomes.length > 0 && (
                                <div className="flex items-center gap-1 text-[#189bff]">
                                    <TrendingUp size={11} strokeWidth={2.5} />
                                    <span>{sortedOutcomes.length}</span>
                                </div>
                            )}
                        </div>
                    </div>
                </div>
            </div>

            {/* Odds Section */}
            <div className="px-4 pb-4 space-y-1.5">
                <div className="flex gap-2.5">
                    {topOutcomes.map((outcome, idx) => {

                        return (
                            <button
                                key={outcome.id}
                                className={`flex-1 min-w-0 flex flex-col gap-1.5 items-start rounded-md cursor-pointer transition-all duration-200 px-3 py-2.5 bg-grayscale-black hover:bg-primary-blue group active:scale-95 active:bg-blue-pressed`}
                                onClick={() => {
                                    closeDrawer();
                                    // Wait some time
                                    openDrawer('bet', { marketId: market.id, outcomeId: outcome.id });
                                }}
                            >
                                <div className="text-sm font-bold line-clamp-1 pr-3 text-start">
                                    {outcome.name}
                                </div>
                                <div className={`font-bold text-sm mr-2 text-primary-blue group-hover:text-white`}>
                                    <AnimatedOdds
                                        prob={outcome.price}
                                        format={oddsFormat}
                                    />
                                </div>

                            </button>
                        );
                    })}
                </div>
            </div>
        </div>
    );
}
