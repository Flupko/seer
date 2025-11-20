import { toastStyled } from "@/ui/Toast";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { postComment } from "../api";



export const usePostComment = (marketId: string, onSuccessAction: () => void) => {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: postComment,
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['comment', 'marketId', marketId] });
            onSuccessAction();
        },
        onError: () => {
            toastStyled("Something went wrong", { type: "error" });
        }
    });
}