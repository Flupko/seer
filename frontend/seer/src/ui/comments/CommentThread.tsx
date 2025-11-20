"use client";

import { CommentsRes, deleteComment, deleteLike, getComments, postLike, reportComment } from "@/lib/api";
import { Comment, CommentSearch, Metadata } from "@/lib/definitions";
import { getNextPageParamFromMetadata } from "@/lib/meta";
import { commentSearchKey } from "@/lib/queries/commentSearchKey";
import { usePostComment } from "@/lib/queries/usePostComment";
import { useUserQuery } from "@/lib/queries/useUserQuery";
import { useModalStore } from "@/lib/stores/modal";
import { timeSince } from "@/lib/utils/date";
import { InfiniteData, useInfiniteQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Ellipsis, Flag, Heart, Trash } from "lucide-react";
import { AnimatePresence, motion } from "motion/react";
import { useEffect, useMemo, useRef, useState } from "react";
import InfiniteScrollContainer from "../InfiniteScrollContainer";
import Input from "../Input";
import Loader from "../loader/Loader";
import ProfilePicture from "../ProfilePicture";
import { toastStyled } from "../Toast";
import ToolTip from "../ToolTip";


const MAX_DEPTH = 5;


export default function CommentThread({ initialCommentSearch }: { initialCommentSearch: CommentSearch }) {

    // Infinite query react query
    const {
        data,
        isLoading,
        isError,
        fetchNextPage,
        isFetchingNextPage,
        hasNextPage,
    } = useInfiniteQuery({
        queryKey: commentSearchKey(initialCommentSearch),
        queryFn: ({ pageParam = 1 }) => getComments({ ...initialCommentSearch, page: pageParam } as CommentSearch),
        getNextPageParam: (lastPage: CommentsRes) => getNextPageParamFromMetadata(lastPage.metadata),
        initialPageParam: 1,
        staleTime: 5 * 60 * 1000,
    });

    const comments = data?.pages.flatMap((p) => p.comments)

    return (
        <>

            {isLoading && <div className="flex justify-center">
                <Loader size={0.7} />
            </div>}

            <div>
                {/* Render comments here using data */}
                {comments?.map(comment => (
                    <CommentDisplay key={comment.id} comment={comment} />
                ))}
            </div>

            {hasNextPage && (
                <InfiniteScrollContainer
                    onBottomReached={fetchNextPage}>
                    {isFetchingNextPage && <div className="flex justify-center">
                        <Loader size={0.7} />
                    </div>}
                </InfiniteScrollContainer>
            )}
        </>
    )
}


function CommentDisplay({ comment }: { comment: Comment }) {

    const queryClient = useQueryClient();

    const [showReplies, setShowReplies] = useState(false);
    const [showReplyInput, setShowReplyInput] = useState(false);
    const [inputValue, setInputValue] = useState("");

    const [showDropdownReport, setShowDropdownReport] = useState(false);
    const dropDownRef = useRef<HTMLDivElement | null>(null);

    const user = useUserQuery().data;

    useEffect(() => {
        const onDown = (e: MouseEvent | PointerEvent) => {
            if (!dropDownRef.current) return;
            if (!dropDownRef.current.contains(e.target as Node)) setShowDropdownReport(false);
        };
        const onKey = (e: KeyboardEvent) => {
            if (e.key === "Escape") setShowDropdownReport(false);
        };
        document.addEventListener("mousedown", onDown);
        document.addEventListener("pointerdown", onDown);
        document.addEventListener("keydown", onKey);
        return () => {
            document.removeEventListener("mousedown", onDown);
            document.removeEventListener("pointerdown", onDown);
            document.removeEventListener("keydown", onKey);
        };
    }, []);


    const openModal = useModalStore(state => state.openModal);

    const replyCommentSearch: CommentSearch = { marketId: comment.marketId, parentId: comment.id, pageSize: 5, page: 1 };
    const inputValid = inputValue.length >= 3;

    const { mutate: mutateComment, isPending: isCommentPending } = usePostComment(comment.marketId, () => {
        setInputValue("");
        setShowReplyInput(false);
        setShowReplies(true);
    })

    const { mutate: mutateLike, isPending: isLikePending } = useMutation({
        mutationFn: (comment.isLiked ? deleteLike : postLike),
        onMutate: () => {
            comment.isLiked = !comment.isLiked;
            comment.nbLikes += comment.isLiked ? 1 : -1;
        },
        onError: () => {
            comment.isLiked = !comment.isLiked;
            comment.nbLikes += comment.isLiked ? 1 : -1;
            toastStyled("Something went wrong", { type: "error" })
        }
    })

    const threadQueryKey = commentSearchKey({ marketId: comment.marketId, parentId: comment.parentId });

    const { mutate: mutateDeleteComment, isPending: isDeletePending } = useMutation({
        mutationFn: deleteComment,
        // Optimistic update
        onMutate: async (deleteCommentReq) => {

            await queryClient.cancelQueries({ queryKey: threadQueryKey });

            const previousData = queryClient.getQueryData<InfiniteData<CommentsRes, Metadata>>(threadQueryKey);

            if (previousData) {
                const newPages = previousData.pages.map(page => {
                    return {
                        metadata: page.metadata,
                        comments: page.comments.filter(c => c.id !== deleteCommentReq.commentId)
                    }
                });
                const newData = {
                    ...previousData,
                    pages: newPages
                };
                queryClient.setQueryData<InfiniteData<CommentsRes, Metadata>>(threadQueryKey, newData);
            }

            return { previousData };
        },
        onError: (err, variables, context) => {
            if (context?.previousData) {
                queryClient.setQueryData<InfiniteData<CommentsRes, Metadata>>(threadQueryKey, context.previousData);
            }
            toastStyled("Something went wrong", { type: "error" });
        },
        onSuccess: () => {
            toastStyled("Comment successfully deleted", { type: "success" });
        }
    });

    const { mutate: mutateReportComment, isPending: isReportPending } = useMutation({
        mutationFn: reportComment,
        onSuccess: () => {
            comment.isReported = true;
            toastStyled("Comment reported", { type: "success" });
        },
        onError: () => {
            toastStyled("Something went wrong", { type: "error" });
        }
    });

    const handleClickReply = () => {
        if (!user) {
            openModal("auth", { selectedTab: "login" });
            return;
        }
        setShowReplyInput(!showReplyInput);
    }

    const handleClickTooltipDropdown = () => {
        if (!user) {
            openModal("auth", { selectedTab: "login" });
            return;
        }
        setShowDropdownReport(!showDropdownReport);
    }

    const handleDeleteComment = () => {
        setShowDropdownReport(false);
        mutateDeleteComment({ commentId: comment.id });
    }

    const handleReportComment = () => {
        if (comment.isReported) {
            toastStyled("Comment already reported", { type: "success" });
            return;
        }
        setShowDropdownReport(false);
        mutateReportComment({ commentId: comment.id });
    }

    const isCommentWriter = user?.id === comment.user.id;

    const timeSinceComment = useMemo(() => timeSince(new Date(comment.createdAt)), [comment]);

    return (
        <div key={comment.id} className="flex mt-6 w-full relative">
            <div className="h-fit">
                <ProfilePicture key={comment.user.profileImageKey} size={40} />
            </div>

            <div className="absolute top-0 right-0" ref={dropDownRef}>
                <ToolTip Icon={Ellipsis} onClick={handleClickTooltipDropdown} />
                <AnimatePresence>
                    {showDropdownReport &&

                        <motion.div

                            initial={{ opacity: 0, y: -5 }}
                            animate={{ opacity: 1, y: 0 }}
                            exit={{ opacity: 0, y: -5 }}
                            transition={{ duration: 0.15, ease: 'easeInOut' }}
                            className="absolute right-0 mt-1 w-28 bg-gray-800 rounded-lg overflow-hidden border border-gray-600 z-10 font-medium">
                            <button disabled={isCommentWriter || isReportPending} className={`w-full text-left px-4 h-11 text-sm ${(isCommentWriter) ? "text-gray-300" : "text-gray-100 hover:bg-gray-700 cursor-pointer"} flex items-center transition-colors duration-200`}
                                onClick={handleReportComment}>
                                <Flag className="h-3.5 w-3.5 mr-2 stroke-1.5" />
                                Report
                            </button>


                            {isCommentWriter &&
                                <button className="w-full text-left px-4 h-11 cursor-pointer text-sm text-gray-100 hover:bg-gray-700 flex items-center transition-colors duration-200"
                                    onClick={handleDeleteComment}
                                    disabled={isDeletePending}>
                                    <Trash className="h-3.5 w-3.5 mr-2 stroke-1.5" />
                                    Delete
                                </button>}


                        </motion.div>

                    }
                </AnimatePresence>
            </div>

            <div className="flex ml-4 flex-col w-[calc(100%-58px)]">

                {/* Username and post date */}
                <div className="flex items-center w-full justify-between gap-3">
                    <div className="flex items-center whitespace-nowrap overflow-hidden text-ellipsis pr-8 gap-1">
                        <div className="flex gap-2 items-center">
                            <span className="text-sm font-bold text-white hover:underline whitespace-nowrap overflow-hidden text-ellipsis">
                                {comment.user.username}
                            </span>
                            {/* DOT SEPARATOR */}
                            <div className="w-1 h-1 bg-gray-500 rounded-full flex-shrink-0 mt-1" />
                            <span className="text-sm font-medium text-gray-400 whitespace-nowrap overflow-hidden text-ellipsis">
                                {timeSinceComment}
                            </span>
                        </div>
                    </div>
                </div>


                <div className="flex mt-2 mb-3 text-sm font-semibold leading-5 break-before-avoid break-all">
                    {comment.content}
                </div>

                {/* Likes and reply */}
                <div className="flex">
                    <button disabled={isLikePending} className={`text-xs flex items-center font-medium w-8 cursor-pointer group`} onClick={() => mutateLike({ commentId: comment.id })}>
                        <Heart className={`inline-block h-4 w-4 stroke-3 ${comment.isLiked ? "fill-current text-orange-500" : "text-gray-400"} group-active:scale-80 duration-200`} />
                        <span className="font-semibold ml-1.5 text-gray-400">{comment.nbLikes}</span>
                    </button>

                    {comment.depth < MAX_DEPTH && <button className="text-xs ml-3 font-bold text-gray-400 w-fit cursor-pointer" onClick={handleClickReply}>
                        Reply
                    </button>}

                </div>

                {/* Reply input */}
                {showReplyInput &&
                    <div className="mt-3.5">
                        <Input border="border-gray-600" bg="bg-transparent" autoFocus={true}
                            value={inputValue} onChange={(e) => setInputValue(e.target.value)}
                            rightEl={<button disabled={!inputValid || isCommentPending} className={`text-sm font-semibold text-primary-blue ${(inputValid && !isCommentPending) ? "cursor-pointer" : "opacity-50"} transition-all`}
                                onClick={() => mutateComment({ marketId: comment.marketId, parentId: comment.id, content: inputValue })}>Post</button>} maxLength={1000} />
                    </div>

                }

                {/* Show replies button */}
                {comment.nbReplies > 0 &&
                    <button className="text-xs mt-3 font-bold text-gray-400 hover:text-gray-300 w-fit cursor-pointer" onClick={() => setShowReplies(!showReplies)}>
                        {showReplies ? "Hide" : "Show"} {comment.nbReplies} {comment.nbReplies === 1 ? "reply" : "replies"}
                    </button>
                }

                {/* Replies */}
                <div>
                    {showReplies && <CommentThread initialCommentSearch={replyCommentSearch} />}
                </div>


            </div>
        </div>
    )
}   