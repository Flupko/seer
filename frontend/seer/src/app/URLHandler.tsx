"use client";

import { useModalStore } from "@/lib/stores/modal";
import { useRouter, useSearchParams } from "next/navigation";
import { useEffect } from "react";
import { toast } from "react-toastify";

export default function URLHandler() {

  const searchParams = useSearchParams();
  const router = useRouter();

  const openModal = useModalStore((state) => state.openModal);

  useEffect(() => {
    const error = searchParams.get("error");
    const show = searchParams.get("show");

    if (error) {
      const errorMessage = decodeURIComponent(error);
      toast.error(errorMessage);

      // Clean URL after showing toast, we remove the error part
      const newParams = new URLSearchParams(searchParams);
      newParams.delete("error");
      router.replace(`/?${newParams.toString()}`, { scroll: false });
    }

    // Handle profile completion modal
    if (show === "profile_completion") {
      openModal("profileCompletion");

      // Clean URL after opening modal, we remove the show part
      const newParams = new URLSearchParams(searchParams);
      newParams.delete("show");
      router.replace(`/?${newParams.toString()}`, { scroll: false });
    }
  }, [searchParams, router, openModal]);

  return null; //  component does not render anything
}
