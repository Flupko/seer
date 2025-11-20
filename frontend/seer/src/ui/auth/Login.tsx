"use client"

import Button from "@/ui/Button";
import { SubmitHandler, useForm } from "react-hook-form";
import Password from "../form/Password";
import Providers from "./Providers";

import { useWebSocket } from "@/app/WsProvider";
import * as api from "@/lib/api";
import { LoginFormValues, LoginSchema } from "@/lib/definitions";
import { useModalStore } from "@/lib/stores/modal";
import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { motion } from "motion/react";
import { useRouter } from "next/navigation";
import FormField from "../form/FormField";


export default function Login() {

    const router = useRouter();

    const ws = useWebSocket();

    const {
        register,
        handleSubmit,
        formState: { errors },
        setError,
    } = useForm<LoginFormValues>({
        resolver: zodResolver(LoginSchema), // Apply the zodResolver
        mode: "onBlur", // Validate on blur
    });

    const queryClient = useQueryClient();
    const closeModal = useModalStore((state) => state.closeModal);


    const mutation = useMutation({
        mutationFn: api.login,
        onSuccess: () => {
            ws?.disonnect();
            queryClient.invalidateQueries({ queryKey: ['user'] });
            router.push("/");
            closeModal();
        },
        onError: (error: api.APIError) => {
            if (error.errors) {
                error.errors.forEach(({ field, message }) => {
                    if (field in LoginSchema.shape) {
                        setError(field as keyof LoginFormValues, { message });
                    }
                });
            }

            if (error.message) {
                setError("root", { message: error.message });
            }
        },
    });

    const onSubmit: SubmitHandler<LoginFormValues> = data => {
        mutation.mutate(data)
    };


    return (
        <div className="flex flex-col gap-8">

            <form className="flex flex-col gap-10" onSubmit={handleSubmit(onSubmit)}>
                <div className="flex flex-col gap-5">

                    <FormField
                        name="login"
                        label="Email or Username*"
                        register={register}
                        error={errors.login}
                        placeholder="Enter Email or Username"
                        type="text"
                    />

                    <Password
                        name="password"
                        label="Password*"
                        register={register}
                        error={errors.password}
                        placeholder="Enter Password"
                    />


                </div>

                {errors.root &&
                    <motion.div className="text-sm font-semibold text-error block text-center" initial={{ opacity: 0 }} animate={{ opacity: 1 }} transition={{ duration: 0.3, ease: "easeIn" }}>
                        {errors.root.message}
                    </motion.div>}


                <Button bg="bg-primary-blue" width="full" height="large" type="submit" isLoading={mutation.isPending}>
                    Login
                </Button>
            </form>


            <Providers />

        </div>)
}

