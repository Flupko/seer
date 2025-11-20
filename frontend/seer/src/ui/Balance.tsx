import { Currency } from "@/lib/definitions";
import { useBalanceQuery } from "@/lib/queries/useBalanceQuery";
import NumberFlow from "@number-flow/react";

export default function Balance({ currency, ...rest }: { currency: Currency } & React.HTMLAttributes<HTMLDivElement>) {
    const { data: balance } = useBalanceQuery(currency);
    if (!balance) return null;

    return (
        <div className="flex items-center gap-3" {...rest}>
            <div className="flex flex-col items-center">
                <span className="text-gray-400 font-medium text-xs">
                    Balance
                </span>
                <span className="font-bold text-sm text-[#43c773] -translate-y-[1px]">
                    <NumberFlow locales={"en-US"}
                        className="tracking-tighter"
                        value={balance.balance.toNumber()}
                        format={{ style: 'currency', currency: 'USD', minimumFractionDigits: 2, maximumFractionDigits: 2 }} />
                </span>
            </div>

            <div className="flex justify-center lg:w-23.5 w-12 shrink-0">
                <button
                    className="bg-primary-blue px-4 font-bold rounded-md h-9 text-[13px] shrink-0 hover:brightness-120 hover:cursor-pointer transition-all active:scale-95 duration-150">
                    Deposit
                </button>
            </div>

        </div>
    );
}
