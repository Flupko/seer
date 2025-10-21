"use client"

import Button from "@/ui/Button";
import { SubmitHandler, useForm } from "react-hook-form";

import * as api from "@/lib/api";
import { ChangePasswordFormValues, ChangePasswordPayloadSchema, ChangePasswordSchema } from "@/lib/definitions";
import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation } from "@tanstack/react-query";
import { motion } from "motion/react";
import { useRouter } from "next/navigation";
import Password from "../form/Password";


export default function ChangePasswordModal() {

    const router = useRouter();

    const {
        register,
        handleSubmit,
        formState: { errors },
        setError,
    } = useForm<ChangePasswordFormValues>({
        resolver: zodResolver(ChangePasswordSchema), // Apply the zodResolver
        mode: "onBlur", // Validate on blur
    });

    const mutation = useMutation({
        mutationFn: api.changePassword,
        onSuccess: () => {
            window.location.replace('/');
        },
        onError: (error: api.APIError) => {
            if (error.errors) {
                error.errors.forEach(({ field, message }) => {
                    if (field in ChangePasswordSchema.shape) {
                        setError(field as keyof ChangePasswordFormValues, { message });
                    }
                });
            }

            if (error.message) {
                setError("root", { message: error.message });
            }
        },
    });

    const onSubmit: SubmitHandler<ChangePasswordFormValues> = data => {
        const payload = ChangePasswordPayloadSchema.parse(data);
        mutation.mutate(payload)
    };


    return (

        <div className="flex flex-col gap-12 px-7.5 lg:px-9.5 pb-8 md:pt-11 pt-6 w-full">

            <div className="flex flex-col gap-5">
                <h1 className="text-2xl font-extrabold">Set Password</h1>
                <p className="text-sm text-white mt-1">Setting your password will log you out from all your sessions.</p>
            </div>


            <form className="flex flex-col gap-15" onSubmit={handleSubmit(onSubmit)}>
                <div className="flex flex-col gap-12">

                    <Password name="currentPassword"
                        label="Current Password*"
                        register={register}
                        error={errors.currentPassword}
                        placeholder="Enter Current Password"
                    />

                    <div className="flex flex-col gap-5">
                        <Password name="newPassword"
                            label="New Password*"
                            register={register}
                            error={errors.newPassword}
                            placeholder="Enter New Password"
                        />

                        <Password name="confirmNewPassword"
                            label="Confirm New Password*"
                            register={register}
                            error={errors.confirmNewPassword}
                            placeholder="Confirm New Password"
                        />
                    </div>



                </div>

                {errors.root &&
                    <motion.div className="text-sm font-semibold text-error block text-center" initial={{ opacity: 0 }} animate={{ opacity: 1 }} transition={{ duration: 0.3, ease: "easeIn" }}>
                        {errors.root.message}
                    </motion.div>}


                <Button bg="bg-neon-blue" width="full" height="large" type="submit" isLoading={mutation.isPending}>
                    Change Password
                </Button>
            </form>


        </div>)
}

