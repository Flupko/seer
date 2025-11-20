import { useUpdateSearchParams } from "@/lib/hooks/useUpdateSearchParams";
import { formatOdds } from "@/lib/odds";
import { usePrefs } from "@/lib/stores/prefs";
import { BetUpdate } from "@/socket/messages";
import { TableCell, TableHead, TableHeading } from "@/ui/Table";
import { HatGlasses } from "lucide-react";
import { AnimatePresence, motion } from "motion/react";
import Image from "next/image";
import Link from "next/link";

export default function BetsLiveTable({ bets, parity }: { bets: BetUpdate[], parity: number }) {

    const oddsFormat = usePrefs((s) => s.oddsFormat);
    const { setParams } = useUpdateSearchParams();

    return (
        <table className="w-full">
            <TableHead className=" bg-grayscale-black z-10 relative [box-shadow:inset_0_-1px_0_theme(colors.gray.600)]">
                <TableHeading className="hidden sm:table-cell">User</TableHeading>
                <TableHeading>Market</TableHeading>
                <TableHeading className="hidden sm:table-cell">Outcome</TableHeading>
                <TableHeading>Wager</TableHeading>
                <TableHeading className="hidden sm:table-cell">Odd</TableHeading>
            </TableHead>

            {/* Enable layout animation on the container that reflows */}

            <motion.tbody layout>
                <AnimatePresence initial={false} mode="sync">
                    {bets.map((bet, index) => (
                        // Animate position changes and mount/unmount
                        <motion.tr
                            layout
                            key={bet.id}
                            initial={{ y: -49 }}
                            animate={{ opacity: 1, y: 0 }}
                            exit={{ opacity: 0, y: 52 }}
                            transition={{ type: "spring", stiffness: 300, damping: 30 }}
                            className="hover:bg-gray-900 children:w-full select-none"
                        >
                            <TableCell className="w-[15%] hidden sm:table-cell" current={index % 2 === parity}>
                                {bet.user ?
                                    <span className="line-clamp-1 hover:text-gray-400 duration-100 w-fit active:scale-95 cursor-pointer"
                                        onClick={() => setParams({ show: 'user', username: bet.user?.username })}>
                                        {bet.user.username}
                                    </span> :
                                    <span className="text-gray-400 flex items-center gap-2">
                                        <HatGlasses className="w-4 h-4" strokeWidth={1.5} />
                                        Hidden
                                    </span>
                                }
                            </TableCell>

                            <TableCell className="w-2/3 sm:w-[30%]" current={index % 2 === parity}>
                                <BetTableLink href={`/market/${bet.marketID}`}>
                                    {bet.marketName}
                                </BetTableLink>
                            </TableCell>

                            <TableCell className="w-[25%] hidden sm:table-cell" current={index % 2 === parity}>
                                <BetTableLink href={`/market/${bet.marketID}`}>
                                    {bet.outcomeName}
                                </BetTableLink>
                            </TableCell>

                            <TableCell className="w-1/3 sm:w-[15%]" current={index % 2 === parity}>

                                <div className="flex items-center gap-2">
                                    <Image src={"/icons/dollar.svg"} alt="Dollar" width={16} height={16} />
                                    {bet.wager.toFixed(2)}
                                </div>

                            </TableCell>

                            <TableCell className="w-[15%] hidden sm:table-cell" current={index % 2 === parity}>
                                {formatOdds(bet.avgPrice, oddsFormat)}
                            </TableCell>
                        </motion.tr>
                    ))}
                </AnimatePresence>
            </motion.tbody>
        </table>
    )
}

function BetTableLink({ href, children }: { href: string, children: React.ReactNode }) {
    return (
        <Link href={href} className="line-clamp-1 hover:text-gray-400 duration-200 w-fit active:scale-95">
            {children}
        </Link>
    )
}