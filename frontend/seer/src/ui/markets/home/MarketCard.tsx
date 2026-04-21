import { useWebSocket } from "@/app/WsProvider";
import { getMarketById } from "@/lib/api";
import { MarketView, Outcome } from "@/lib/definitions"; // Import from your types file
import { useOdds } from "@/lib/hooks/useOdds";
import { isMarketActive, isMarketPending } from "@/lib/markets";
import { useModalStore } from "@/lib/stores/modal";
import { CheckYes } from "@/ui/Checks";
import { ArrowOdds } from "@/ui/odds/ArrowOdds";
import TextFade from "@/ui/TextFade";
import NumberFlow from "@number-flow/react";
import { useQuery } from "@tanstack/react-query";
import { Bookmark, CheckCircle2, CircleX, ClockFading } from "lucide-react";
import Image from "next/image";
import { useRouter } from "next/navigation";
import { useEffect } from "react";

export default function MarketCard({ marketInitial }: { marketInitial: MarketView }) {

    const { oddsFormat } = useOdds();

    const openModal = useModalStore(state => state.openModal);

    const { data: market } = useQuery({
        queryKey: ['market', marketInitial.id],
        queryFn: () => getMarketById(marketInitial.id),
        initialData: marketInitial,
        staleTime: Infinity,
    })

    const router = useRouter();


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
            return b.priceYes.minus(a.priceYes).toNumber();
        }
        return a.position - b.position;
    });

    // Top 3 outcomes for main display
    const topOutcomes = sortedOutcomes.slice(0, 2);

    return (
        <div
            className="rounded-lg py-4 h-[215px] bg-gray-800 hover:backdrop-brightness-50 border border-gray-700 px-4 transition-all duration-200 flex justify-between flex-col"
        >
            {/* Header */}
            <div className="group hover:cursor-pointer">
                <div className="flex gap-4 items-center">
                    {/* Compact Thumbnail */}
                    {market.imgKey && (
                        <div className="flex-shrink-0 w-12 h-12 rounded-lg overflow-hidden group-hover:scale-105 transition-all duration-300">
                            <Image
                                src={market.imgKey}
                                alt={market.name}
                                width={250}
                                height={250}
                                className="object-cover w-full h-full"
                            />
                        </div>
                    )}

                    {/* Market Info */}
                    <div className="flex-1 min-w-0 items-center">

                        <h3 className="text-[15px] font-bold group-hover:text-gray-400 text-white leading-6 line-clamp-2 transition-colors" onClick={() => router.push(`/market/${market.id}`)}>
                            {market.name}
                        </h3>

                        {/* Metadata Row */}
                        {/* <div className="flex items-center gap-2.5 text-gray-400 font-medium">

                            {market.closeTime && market.status !== "resolved" && (
                                <div className={`flex items-center text-xs font-normal gap-1 text-gray-400`}>
                                    <Clock size={11} strokeWidth={2} />
                                    <span className="line-clamp-1">{formatDateTime(market.closeTime)}</span>
                                </div>
                            )}

                            {market.status === "resolved" && market.resolution && (
                                <div className={`flex items-center text-xs font-normal gap-1 text-gray-400`}>
                                    <Clock size={11} strokeWidth={2} />
                                    <span className="line-clamp-1">Resolved on {market.resolution.createdAt.toLocaleDateString('en-US', {
                                        month: 'short',
                                        day: 'numeric',
                                        hour: '2-digit',
                                        minute: '2-digit',
                                        hour12: false
                                    })
                                    }</span>
                                </div>
                            )}

                        </div> */}


                    </div>
                </div>
            </div>

            {/* Odds Section */}

            {market.status === "resolved" && market.resolution && (
                <WinningOutcome winningOutcome={market.outcomes.find(o => o.id === market.resolution!.winningOutcomeId)!} />

            )}

            {!market.isBinary && market.status !== "resolved" && (


                <div className="flex flex-col gap-2.5">
                    {topOutcomes.map((outcome, idx) => {


                        return (
                            <div
                                key={outcome.id}
                                className={`flex-1 flex justify-between
                                    ${market.status === 'active' && "cursor-pointer group"}
                                    ${market.status === "resolved" && market.resolution?.winningOutcomeId !== outcome.id && "brightness-40"}
                                    transition-all duration-200 pointer-events-auto`}
                                onClick={() => {
                                    console.log('open bet modal for outcome', outcome.id);
                                    if (!isMarketActive(market)) return;

                                    openModal('bet', { marketId: market.id, initialOutcomeId: outcome.id, initialSide: 'y' });
                                }}
                            >
                                <div className="text-sm text-start flex items-center gap-1.5 min-w-0 mr-2" >
                                    <div
                                        className="font-medium group-hover:text-gray-400 transition-colors duration-200 overflow-hidden"
                                    >
                                        <TextFade>
                                            {outcome.name}
                                        </TextFade>

                                    </div>

                                    {market.status === "resolved" && (market.resolution?.winningOutcomeId === outcome.id ?
                                        <span><CheckCircle2 className="text-success" size={14} /></span> :
                                        <span><CircleX className="text-red-400" size={14} /></span>)
                                    }

                                </div>

                                <div className="flex items-center gap-4">
                                    <div className={`font-bold text-lg leading-7.5 text-white group-hover:text-gray-400 transition-colors duration-200`}>
                                        <ArrowOdds
                                            prob={outcome.priceYes}
                                            format={oddsFormat}
                                        />
                                    </div>

                                    <div className="flex gap-1">
                                        {/* YES / NO buttons, */}


                                        <div className="flex items-center h-7.5 rounded-lg text-[13px] font-semibold bg-gradient-to-r from-yes/90 to-no/90 relative group-hover:brightness-110 active:scale-95 duration-100"  >
                                            <button className="text-center text-yes-text cursor-pointer pl-2.5 pr-4 h-7.5"
                                                onClick={(e) => {
                                                    e.stopPropagation();
                                                    if (!isMarketActive(market)) return;
                                                    openModal('bet', { marketId: market.id, initialOutcomeId: outcome.id, initialSide: 'y' });
                                                }}>Yes</button>

                                            <button className="text-center text-no-text cursor-pointer pr-2.5 pl-4 h-7.5"
                                                onClick={(e) => {
                                                    e.stopPropagation();
                                                    if (!isMarketActive(market)) return;
                                                    openModal('bet', { marketId: market.id, initialOutcomeId: outcome.id, initialSide: 'n' });
                                                }}>No</button>
                                            <span className="absolute text-gray-400 right-11">/</span>
                                        </div>

                                    </div>

                                </div>


                            </div>
                        );
                    })}
                </div>
            )}

            {market.isBinary && market.status !== "resolved" && (
                <div className="flex gap-2">
                    <button className={`flex w-full justify-center gap-3 items-center ${isMarketActive(market) ? "hover:text-white hover:bg-yes-neon hover:brightness-110 cursor-pointer active:scale-95" : "brightness-60"} bg-yes text-yes-text rounded-md h-12 transition-all duration-100`}
                        onClick={() => {
                            if (!isMarketActive(market)) return;
                            openModal('bet', { marketId: market.id, initialOutcomeId: market.outcomes[0].id, initialSide: 'y' });
                        }}
                        disabled={!isMarketActive(market)}>
                        <span className="flex items-baseline gap-2">
                            <span className="font-medium text-[15px] line-clamp-1 break-all">{market.outcomes[0].name}</span>
                            <span className="font-bold text-[17px]"><ArrowOdds
                                prob={market.outcomes[0].priceYes}
                                format={oddsFormat}
                            />
                            </span>
                        </span>
                    </button>

                    <button className={`flex w-full justify-center gap-3 items-center ${isMarketActive(market) ? "hover:text-white hover:bg-no-neon hover:brightness-110 cursor-pointer active:scale-95" : "brightness-60"} bg-no text-no-text rounded-md h-12 transition-all duration-100`}
                        onClick={() => {
                            if (!isMarketActive(market)) return;
                            openModal('bet', { marketId: market.id, initialOutcomeId: market.outcomes[1].id, initialSide: 'y' });
                        }}
                        disabled={!isMarketActive(market)}>
                        <span className="flex items-baseline gap-2">
                            <span className="font-medium text-[15px]">{market.outcomes[1].name}</span>
                            <span className="font-bold text-[17px]"><ArrowOdds
                                prob={market.outcomes[1].priceYes}
                                format={oddsFormat}
                            />
                            </span>
                        </span>
                    </button>

                </div >
            )}




            <div className="flex justify-between items-center font-normal">

                <div className="flex items-center gap-3 text-[13px] font-medium">
                    {market.totalVolume && (
                        <div className="flex gap-3 text-gray-400 items-center">
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

                    {isMarketPending(market) && (
                        <div className="flex gap-1.5 text-gray-400 items-center rounded-full bg-gray-700/60 px-2.5 h-6">
                            <ClockFading className="w-3 h-3" />
                            <span className="font-medium">
                                Pending
                            </span>
                        </div>
                    )}


                </div>



                <span>
                    <Bookmark size={18} className="text-gray-400" />
                </span>
            </div>


        </div >
    );
}

function WinningOutcome({ winningOutcome }: { winningOutcome: Outcome }) {
    return (
        <div className="flex w-full justify-center gap-3 items-center bg-gray-700  rounded-md h-12 transition-all duration-100">
            <span className="flex items-baseline gap-2">
                <span className="font-bold text-[15px] line-clamp-1 break-all">
                    {winningOutcome.name}
                </span>
                <CheckYes size="w-2.5 h-2.5" className="text-gray-700 bg-success" />
            </span>
        </div>
    )
}




