import { MarketView } from "@/lib/definitions";
import { usePostComment } from "@/lib/queries/usePostComment";
import { useUserQuery } from "@/lib/queries/useUserQuery";
import { useState } from "react";
import Input from "../Input";
import CommentThread from "./CommentThread";


export default function CommentSection({ market }: { market: MarketView }) {

    const [inputValue, setInputValue] = useState("");
    const inputValid = inputValue.length >= 3

    const { mutate, isSuccess, isPending } = usePostComment(market.id, () => {
        setInputValue("");
    })

    const { data: user } = useUserQuery();

    return (
        <div>
            <div className="mb-8">
                <Input border="border-gray-600" bg="bg-transparent" placeholder="Add a comment"
                    value={inputValue} onChange={(e) => setInputValue(e.target.value)}
                    rightEl={
                        <button disabled={!inputValid || isPending} className={`text-sm font-semibold text-primary-blue ${(inputValid && !isPending) ? "cursor-pointer" : "opacity-50"} transition-all`}
                            onClick={() => mutate({ marketId: market.id, content: inputValue })}>Post</button>} maxLength={1000} />
            </div>
            <CommentThread initialCommentSearch={{ marketId: market.id, pageSize: 10, page: 1 }} />
        </div>
    )
}