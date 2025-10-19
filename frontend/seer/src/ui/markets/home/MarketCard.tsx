import { MarketView } from "@/lib/definitions"; // Import from your types file
import { ChevronDown, Clock, TrendingUp } from "lucide-react";
import { AnimatePresence, motion } from "motion/react";
import Image from "next/image";
import { useState } from "react";

export default function MarketCard({ market }: { market: MarketView }) {
    const [showAllOutcomes, setShowAllOutcomes] = useState(false);

    // Sort outcomes based on outcomeSort preference
    const sortedOutcomes = [...market.outcomes].sort((a, b) => {
        if (market.outcomeSort === 'price') {
            return b.probPPM - a.probPPM;
        }
        return a.position - b.position;
    });

    // Show only active outcomes
    const activeOutcomes = sortedOutcomes.filter(o => o.active);

    // Top 2 outcomes for main display
    const topOutcomes = activeOutcomes.slice(0, 2);
    const remainingOutcomes = activeOutcomes.slice(2);

    const getTimeRemaining = (dateString: string) => {
        const closeDate = new Date(dateString);
        const now = new Date();
        const diff = closeDate.getTime() - now.getTime();

        if (diff < 0) return { text: "Closed", urgent: false };

        const days = Math.floor(diff / (1000 * 60 * 60 * 24));
        const hours = Math.floor((diff % (1000 * 60 * 60 * 24)) / (1000 * 60 * 60));

        if (days > 0) return { text: `${days}d ${hours}h`, urgent: false };
        if (hours > 3) return { text: `${hours}h`, urgent: false };
        return { text: hours > 0 ? `${hours}h left` : "Ending soon", urgent: true };
    };

    // Convert probPPM to decimal odds
    const calculateOdds = (probPPM: number) => {
        if (probPPM === 0) return "0.00";
        const odds = 1000000 / probPPM;
        return odds.toFixed(2);
    };

    const timeInfo = market.closeTime ? getTimeRemaining(market.closeTime) : null;

    return (
        <motion.div
            className="bg-gray-800 group rounded-lg cursor-pointer overflow-hidden hover:bg-gray-700/80"
            whileHover={{ y: -0.8 }}
            transition={{ ease: "linear", duration: 0.085 }}
        >
            {/* Header */}
            <div className="p-4 mb-1">
                <div className="flex gap-3 items-start">
                    {/* Compact Thumbnail */}
                    {market.imgKey && (
                        <div className="flex-shrink-0 w-12 h-12 rounded overflow-hidden bg-gray-900">
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
                        <div className="flex items-start gap-2 mb-1.5">
                            <h3 className="text-sm font-bold text-white leading-tight line-clamp-2 mb-2 group-hover:text-neon-blue transition-colors">
                                {market.name}
                            </h3>
                        </div>

                        {/* Metadata Row */}
                        <div className="flex items-center gap-3 text-gray-400 text-[11px] font-bold">
                            {market.categories.length > 0 && (
                                <motion.span
                                    whileHover={{ scale: 1.05 }}
                                    className="text-neon-blue bg-neon-blue/10 px-2 py-0.5 rounded-full"
                                >
                                    {market.categories[0].label}
                                </motion.span>
                            )}
                            {timeInfo && (
                                <div className={`flex items-center gap-1 ${timeInfo.urgent ? 'text-neon-blue animate-pulse' : 'text-gray-400'
                                    }`}>
                                    <Clock size={11} strokeWidth={2.5} />
                                    <span>{timeInfo.text}</span>
                                </div>
                            )}

                            {activeOutcomes.length > 0 && (
                                <div className="flex items-center gap-1 text-gray-400">
                                    <TrendingUp size={11} strokeWidth={2.5} />
                                    <span>{activeOutcomes.length}</span>
                                </div>
                            )}
                        </div>
                    </div>
                </div>
            </div>

            {/* Odds Section */}
            <div className="px-4 pb-4 space-y-1.5">
                {topOutcomes.map((outcome, idx) => {
                    const odds = calculateOdds(outcome.probPPM);
                    const isTopOutcome = idx === 0;

                    return (
                        <motion.button
                            key={outcome.id}
                            whileHover={{ scale: 1.015 }}
                            whileTap={{ scale: 0.97 }}
                            className={`w-full rounded-lg cursor-pointer transition-all duration-150 p-3 flex items-center justify-between group bg-gray-900 hover:bg-grayscale-black`}
                        >
                            <span className="text-sm font-semibold text-white truncate pr-3">
                                {outcome.name}
                            </span>
                            <div className={`font-numbers text-base font-bold px-4 py-2 rounded ${isTopOutcome
                                ? 'bg-neon-blue text-white'
                                : 'bg-gray-800 text-white'
                                }`}>
                                {odds}
                            </div>

                            {/* <div className={`font-numbers text-lg font-bold px-4 py-1.5 rounded ${isTopOutcome
                                ? 'bg-neon-blue text-white'
                                : 'bg-gray-800 text-white'
                                }`}>
                                {odds}
                            </div> */}
                        </motion.button>
                    );
                })}

                {/* Expandable Outcomes */}
                {remainingOutcomes.length > 0 && (
                    <div>
                        <button
                            onClick={(e) => {
                                e.stopPropagation();
                                setShowAllOutcomes(!showAllOutcomes);
                            }}
                            className="w-full flex items-center justify-center gap-2 text-[11px] font-bold text-gray-400 hover:text-neon-blue transition-colors duration-150 py-2.5 mt-1"
                        >
                            <span>+{remainingOutcomes.length}MORE MARKETS</span>
                            <motion.span
                                animate={{ rotate: showAllOutcomes ? 180 : 0 }}
                                transition={{ duration: 0.15, ease: "easeInOut" }}
                            >
                                <ChevronDown size={14} strokeWidth={2.5} />
                            </motion.span>
                        </button>

                        <AnimatePresence>
                            {showAllOutcomes && (
                                <motion.div
                                    initial={{ height: 0, opacity: 0 }}
                                    animate={{ height: "auto", opacity: 1 }}
                                    exit={{ height: 0, opacity: 0 }}
                                    transition={{ duration: 0.2, ease: "easeInOut" }}
                                    className="overflow-hidden"
                                >
                                    <div className="mt-1 max-h-60 overflow-y-auto space-y-1.5 pr-1 scrollbar-thin scrollbar-thumb-gray-600 scrollbar-track-transparent">
                                        {remainingOutcomes.map((outcome) => {
                                            const odds = calculateOdds(outcome.probPPM);

                                            return (
                                                <motion.button
                                                    key={outcome.id}
                                                    whileHover={{ scale: 1.015 }}
                                                    whileTap={{ scale: 0.985 }}
                                                    className="w-full bg-gray-900 hover:bg-gray-700 rounded-md transition-all duration-150 p-2.5 flex items-center justify-between group"
                                                >
                                                    <span className="text-xs font-semibold text-white truncate pr-2">
                                                        {outcome.name}
                                                    </span>
                                                    <div className="font-numbers text-base font-bold text-gray-200 bg-gray-800 group-hover:bg-gray-600 px-3 py-1 rounded">
                                                        {odds}
                                                    </div>
                                                </motion.button>
                                            );
                                        })}
                                    </div>
                                </motion.div>
                            )}
                        </AnimatePresence>
                    </div>
                )}
            </div>
        </motion.div>
    );
}
