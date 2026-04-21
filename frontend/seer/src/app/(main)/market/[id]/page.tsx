"use client";

import { getMarketById } from "@/lib/api";
import { isMarketActive, isMarketPending } from "@/lib/markets";
import { useUserQuery } from "@/lib/queries/useUserQuery";
import { useModalStore } from "@/lib/stores/modal";
import { usePrefs } from "@/lib/stores/prefs";
import { formatDateTime } from "@/lib/utils/date";
import { CheckNo, CheckYes, Time } from "@/ui/Checks";
import CommentSection from "@/ui/comments/CommentSection";
import MarketBetSection from "@/ui/markets/details/MarketBetSection";
import { MarketDescription } from "@/ui/markets/details/MarketDescription";
import PriceCharts from "@/ui/markets/details/PriceCharts";
import { AnimatedOdds } from "@/ui/odds/AnimatedOdds";
import { ArrowOdds } from "@/ui/odds/ArrowOdds";
import { toastStyled } from "@/ui/Toast";
import NumberFlow from "@number-flow/react";
import { useQuery } from "@tanstack/react-query";
import { Bookmark, Clock, Copy, MessageCircle, TrendingUp } from "lucide-react";
import Image from "next/image";
import { use } from "react";

export default function MarketPage({
    params,
}: {
    params: Promise<{ id: string }>
}) {

    const { id } = use(params)
    const { data: market } = useQuery({
        queryKey: ['market', id],
        queryFn: () => getMarketById(id),
        staleTime: Infinity,
    });

    const openModal = useModalStore(state => state.openModal);

    const oddsFormat = usePrefs(state => state.oddsFormat);
    const { data: user } = useUserQuery();

    if (!market) return

    const timeInfo = market.closeTime ? formatDateTime(market.closeTime) : null;

    return (

        <div className="space-y-6">

            <div className="transition-all overflow-hidden">
                <div className="relative">
                    <div className="flex justify-between">
                        <div className="flex flex-col sm:flex-row gap-4 sm:gap-6 sm:items-center">
                            {market.imgKey && (
                                <div className="flex-shrink-0 h-16 w-16 md:w-18 md:h-18 rounded-lg overflow-hidden">
                                    <Image
                                        src={market.imgKey}
                                        alt={market.name}
                                        width={250}
                                        height={250}
                                        className="object-cover w-full h-full"
                                    />
                                </div>
                            )}

                            <h2 className="text-2xl md:text-[28px] font-bold text-white tracking-tight line-clamp-2 transition-colors">
                                {market?.name}
                            </h2>

                        </div>

                        {/* Share */}
                        <div className="flex absolute right-0 md:static gap-1.5 items-center">

                            <button className="w-8 h-8 flex items-center justify-center cursor-pointer hover:bg-gray-700 rounded-lg text-sm font-medium active:scale-95 transition-all">
                                <MessageCircle className="w-4.5 h-4.5 text-gray-100" />
                            </button>

                            <button className="w-8 h-8 flex items-center justify-center cursor-pointer hover:bg-gray-700 rounded-lg text-sm font-medium active:scale-95 transition-all">
                                <Bookmark className="w-4.5 h-4.5 text-gray-100" />
                            </button>

                            {/* Copy to clipboard */}
                            <button
                                onClick={() => {
                                    const shareUrl = `${window.location.origin}/market/${market.id}`;
                                    navigator.clipboard.writeText(shareUrl);
                                    toastStyled("Market link copied to clipboard!", { type: "success", autoClose: 1500 });
                                }}
                                className="w-8 h-8 flex items-center justify-center cursor-pointer hover:bg-gray-700 rounded-lg text-sm font-medium active:scale-95 transition-all">
                                <Copy className="w-4.5 h-4.5 text-gray-100" />

                            </button>
                        </div>


                    </div>


                    {/* Metadata Row */}
                    <div className="flex items-center gap-4 text-gray-400 font-medium text-xs md:text-sm flex-wrap mt-6">

                        <div className="flex gap-3 items-center">
                            <TrendingUp strokeWidth={2.5} className="mt-0.5 w-4 h-4 md:w-4.5 md:h-4.5" />

                            <span className="flex gap-1.5 items-center">

                                <NumberFlow locales={"en-US"}
                                    value={market.totalVolume.toNumber()}
                                    format={{ style: 'currency', currency: 'USD', minimumFractionDigits: 2, maximumFractionDigits: 2 }} />

                                <span>
                                    Vol.
                                </span>
                            </span>
                        </div>

                        {timeInfo && market.status !== "resolved" && (
                            <div className={`flex gap-1.5 items-center`}>
                                <Clock className="w-3.5 h-3.5 md:w-4 md:h-4" strokeWidth={2.5} />
                                <span className="line-clamp-1">{timeInfo}</span>
                            </div>
                        )}

                        {market.status === "resolved" && market.resolution && (
                            <div className={`flex gap-1.5 items-center`}>
                                <Clock className="w-3.5 h-3.5 md:w-4 md:h-4" strokeWidth={2.5} />
                                <span className="line-clamp-1">Resolved on {market.resolution.createdAt.toLocaleDateString('en-US', {
                                    month: 'short',
                                    day: 'numeric',
                                    hour: '2-digit',
                                    minute: '2-digit',
                                    hour12: false
                                })}
                                </span>
                            </div>


                        )}

                    </div>



                    {/* Charts */}

                    <div className="mt-6">
                        <PriceCharts outcomes={market.outcomes} />
                    </div >
                </div>

                {/* Resolution */}
                {market.status === "resolved" && market.resolution && (
                    <div className="mt-8 p-6 md:p-8 bg-grayscale-800 rounded-lg border border-gray-700 flex flex-col gap-4 md:gap-6 items-center">
                        <CheckYes size="w-8.5 h-8.5" padding="p-1.5" className="text-grayscale-black bg-primary-blue" />
                        <p className="text-primary-blue font-bold text-xl">Outcome: {market.outcomes.find(o => o.id === market.resolution!.winningOutcomeId)?.name}</p>
                        <p className="text-gray-400 text-center text-sm">Resolved on {market.resolution.createdAt.toLocaleDateString('en-US', {
                            month: 'short',
                            day: 'numeric',
                            year: 'numeric',
                            hour12: false
                        })} at {market.resolution.createdAt.toLocaleTimeString('en-US', {
                            hour: '2-digit',
                            minute: '2-digit',
                            hour12: false
                        })}.</p>
                    </div>
                )}

                {isMarketPending(market) && (
                    <div className="mt-8 p-6 md:p-8 bg-grayscale-800 rounded-lg border border-gray-700 flex flex-col gap-4 md:gap-6 items-center">
                        <Time size="w-8.5 h-8.5" padding="p-1.5" className="text-grayscale-black bg-primary-blue" />
                        <p className="text-primary-blue font-bold text-xl">Pending Resolution</p>
                        <p className="text-gray-400 text-center text-sm">This market will soon be resolved.</p>
                    </div>
                )}






                <div className="mt-6">
                    <div
                        className="shrink-0 self-stretch h-px w-full bg-gray-700"
                    />

                    <div className="h-10 items-center gap-2 justify-between w-full text-[11px] hidden md:flex">
                        <div className="flex-1">
                            <span className="text-gray-400 font-bold pl-0.5 tracking-wider">OUTCOME</span>
                        </div>
                        <div className="flex gap-1 items-center w-20">
                            <span className="text-gray-400 font-bold tracking-wider">% CHANCE</span>
                        </div>


                        <div className="flex-1">
                        </div>
                    </div>

                    <div
                        className="shrink-0 self-stretch h-px w-full bg-gray-700 hidden md:block"
                    />
                    <div className="flex flex-col">
                        {market.outcomes.map((outcome, idx) => (
                            <div
                                key={outcome.id}
                                className={`flex flex-col md:flex-row gap-6 md:items-center md:justify-between py-3 md:py-2 border-b border-gray-700 pl-0.5
                                    bg-grayscale-black
                                    ${market.status === 'active' && "cursor-pointer"}
                                    transition-all duration-200`}
                            // onClick={() => {
                            //     if (market.status !== 'active') return;
                            //     openDrawer('bet', { marketId: market.id, initialOutcomeId: outcome.id });
                            // }}
                            >
                                <div className="text-[16px] flex-1 shrink text-start flex items-center gap-3">
                                    <span className="line-clamp-1 font-bold">{outcome.name}</span>

                                    {market.status === "resolved" && (market.resolution?.winningOutcomeId === outcome.id ?
                                        <CheckYes size="w-3 h-3" className="text-gray-800 bg-success" /> :
                                        <CheckNo size="w-3 h-3" className="text-gray-800 bg-error" />)
                                    }

                                </div>
                                <div className={`font-bold absolute right-0 md:static flex text-2xl text-primary-blue group-hover:text-white w-20 price-display`} >
                                    <AnimatedOdds prob={outcome.priceYesNormalized} format={"percent"} />
                                </div>


                                {!market.isBinary ? (<div className="flex-1 flex md:justify-end gap-2">
                                    <button className={`flex w-full md:w-34 justify-center gap-2 items-center px-8 md:px-0 bg-yes hover:bg-yes-neon text-yes-text rounded-md h-12  ${isMarketActive(market) ? `hover:brightness-110 cursor-pointer active:scale-95 hover:text-white` : "brightness-60"}  transition-all duration-100`}
                                        disabled={market.status !== 'active'}
                                        onClick={() => {
                                            if (market.status !== 'active') return;
                                            openModal('bet', { marketId: market.id, initialOutcomeId: outcome.id, initialSide: 'y' });
                                        }}>
                                        <span className="flex items-baseline gap-2">
                                            <span className="font-medium text-[15px]">Yes</span>
                                            <span className="font-bold text-lg price-display"><ArrowOdds
                                                prob={outcome.priceYes}
                                                format={oddsFormat}
                                            />
                                            </span>
                                        </span>

                                    </button>
                                    <button className={`flex w-full md:w-34 justify-center gap-2 items-center px-8 md:px-0 bg-no enabled:hover:bg-no-neon text-no-text rounded-md h-12  ${isMarketActive(market) ? `hover:brightness-110 cursor-pointer active:scale-95 hover:text-white` : "brightness-60"}  transition-all duration-100`}
                                        disabled={market.status !== 'active'}
                                        onClick={() => {
                                            if (market.status !== 'active') return;
                                            openModal('bet', { marketId: market.id, initialOutcomeId: outcome.id, initialSide: 'n' });
                                        }}>
                                        <span className="flex items-baseline gap-2">
                                            <span className="font-bold text-[15px]">No</span>
                                            <span className="font-bold text-lg price-display"><ArrowOdds
                                                prob={outcome.priceNo}
                                                format={oddsFormat}
                                            />
                                            </span>
                                        </span>

                                    </button>
                                </div>

                                ) :

                                    <div className="flex-1 flex md:justify-end gap-2">
                                        <button className={`flex w-full md:w-34 justify-center gap-2 items-center px-8 md:px-0 rounded-md h-12 
                                            ${idx === 0 ? `bg-no enabled:hover:bg-no-neon text-no-text` : `bg-yes enabled:hover:bg-yes-neon text-yes-text`}
                                            ${isMarketActive(market) ? `hover:brightness-110 cursor-pointer active:scale-95 hover:text-white` : "brightness-60"} 
                                            transition-all duration-100`}
                                            disabled={market.status !== 'active'}
                                            onClick={() => {
                                                if (market.status !== 'active') return;
                                                openModal('bet', { marketId: market.id, initialOutcomeId: outcome.id, initialSide: 'y' });
                                            }}>
                                            <span className="flex items-baseline gap-2">
                                                <span className="font-bold text-xl price-display"><ArrowOdds
                                                    prob={outcome.priceYes}
                                                    format={oddsFormat}
                                                />
                                                </span>
                                            </span>

                                        </button>
                                    </div>}

                            </div>
                        ))}
                    </div>
                </div>



                {/* Outcomes */}

            </div >

            {user && <MarketBetSection market={market} />}


            <div className="mt-7 md:mt-14">
                <h3 className="text-lg md:text-xl font-bold text-white mb-5">Market Rules</h3>
                <MarketDescription description={market.description} />
            </div>

            <div className="mt-7 md:mt-14">
                <h3 className="text-lg md:text-xl font-bold text-white mb-5">Comments</h3>
                {/* TODO comments */}
                <CommentSection market={market} />

            </div>



        </div >

    );
}