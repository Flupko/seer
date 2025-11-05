"use client"


import { getUserProfile } from "@/lib/api";
import { useUpdateSearchParams } from "@/lib/hooks/useUpdateSearchParams";
import { useQuery } from "@tanstack/react-query";
import { useSearchParams } from "next/navigation";
import { useImperativeHandle } from "react";

import { useModalStore } from "@/lib/stores/modal";
import ProfilePicture from "../ProfilePicture";
import { UserStats } from "./UserStats";


export default function UserInfoModal({ ref }: { ref: { handleClose: () => void } }) {

    const params = useSearchParams();
    const username = params.get("username");


    const closeModal = useModalStore((state) => state.closeModal);

    const handleClose = () => {
        setParams({ show: null, username: null });
    }

    const { setParams } = useUpdateSearchParams();
    useImperativeHandle(ref as any, () => ({
        handleClose
    }));

    const { data: userProfile, isPending } = useQuery({
        queryKey: ['userProfile', username],
        queryFn: () => getUserProfile(username as string),
        staleTime: 5 * 60 * 1000, // 5 minutes
        enabled: !!username,
    })

    if (!userProfile && !isPending) {
        handleClose();
        closeModal();
        return null;
    }

    if (!userProfile || !username) {
        return null;
    }

    return (

        <div className="flex flex-col gap-12 px-7.5 lg:px-9.5 pb-8 md:pt-11 pt-6 w-full">

            <div className="flex flex-col gap-9 items-center">

                <div className="flex flex-col gap-5 items-center">
                    <ProfilePicture size={96} url={userProfile.profileImageKey} />
                    <h1 className="text-2xl font-extrabold">{userProfile.username}</h1>
                </div>

                <UserStats createdAt={userProfile.createdAt} totalWagered={userProfile.totalWagered} />

            </div>

        </div>)
}

