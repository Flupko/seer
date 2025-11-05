import Decimal from "decimal.js";
import DollarIcon from "../../../public/icons/dollar.svg";
import ContainerSmall from "../containers/ContainerSmall";

export function UserStats({ createdAt, totalWagered }: { createdAt: Date, totalWagered: Decimal }) {
    return (
        <ContainerSmall className="w-full grid md:grid-cols-2 md:gap-0 md:px-0">
            <div className="flex justify-between md:justify-start md:flex-col items-center gap-1 border-b md:border-b-0 md:border-r border-gray-700 md:pr-1.5 pb-4 md:pb-0">
                <span className="text-gray-300 block text-sm font-medium mb-1">Join date</span>
                <span className="font-medium text-sm">{createdAt.toLocaleDateString()}</span>
            </div>

            <div className="flex justify-between md:justify-start md:flex-col items-center gap-1 md:pl-1.5 pt-4 md:pt-0">
                <span className="text-gray-300 block text-sm font-medium mb-1">Total Wagered</span>
                <span className="font-medium text-sm flex items-center gap-2">
                    <DollarIcon className="w-4 h-4" />
                    {/* US comma format */}
                    ${totalWagered.toFixed(2).replace(/\B(?=(\d{3})+(?!\d))/g, ",")}
                </span>
            </div>
        </ContainerSmall>
    )
}