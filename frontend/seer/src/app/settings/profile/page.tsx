"use client"

import { useUserQuery } from "@/lib/queries/useUserQuery";
import ContainerSmall from "@/ui/containers/ContainerSmall";
import MainWrapper from "@/ui/MainWrapper";
import ProfilePicture from "@/ui/ProfilePicture";
import { CircleAlert, CircleCheck, Edit, VerifiedIcon } from "lucide-react";
import Image from "next/image";

export default function SettingsPage() {
    const { data } = useUserQuery();
    if (!data) return null;
    const user = data;

    return (
        <div className="border border-gray-600 rounded-md p-6">
            <h1 className="text-lg font-bold mb-5">Profile Settings</h1>

            <div className="flex flex-col gap-6 max-w-xl">
                <ContainerSmall>

                    <div className="flex justify-between items-center">

                        <div className="flex items-center gap-3">
                            <ProfilePicture url={user.profileImageUrl} size={45} />
                            <span className="font-bold text-lg">{user.username}</span>
                        </div>

                        <Edit className="text-gray-100 hover:text-gray-300 cursor-pointer transition-all" size={22}/>

                    </div>
                </ContainerSmall>

                <ContainerSmall>
                    <div className="flex flex-col gap-1">
                        <span className="text-gray-400 block text-sm font-medium mb-1">Email</span>
                        <div className="flex justify-between items-center">
                            <span className="font-medium">{user.email}</span>
                            {user.status === "activated" ? (
                                <CircleCheck className="text-green-500" size={19} />
                            ) : (
                                <CircleAlert className="text-red-500" size={19} />
                            )}
                        </div>
                    </div>
                </ContainerSmall>

                <ContainerSmall>
                    </ContainerSmall>
            </div>
        </div>
    )
}