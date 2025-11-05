"use client";

import { useUpdateSearchParams } from "@/lib/hooks/useUpdateSearchParams";
import { useModalStore } from "@/lib/stores/modal";
import { useSearchParams } from "next/navigation";
import { useEffect } from "react";
import { toast } from "react-toastify";

export default function URLHandler() {

  const searchParams = useSearchParams();
  const { setParams } = useUpdateSearchParams();

  const openModal = useModalStore((state) => state.openModal);

  useEffect(() => {
    const error = searchParams.get("error");
    const show = searchParams.get("show");

    if (error) {
      const errorMessage = decodeURIComponent(error);
      toast.error(errorMessage);

      setParams({ error: null });
    }

    // Handle profile completion modal
    switch (show) {
      case "profile_completion":
        openModal("profileCompletion");

        // Clean URL after opening modal
        setParams({ show: null });
        break;

      case "user":
        const username = searchParams.get("username");
        if (username) {
          openModal("user", { username });
        }
        break;

      default:
        break;
    }

  }, [searchParams, openModal, setParams]);

  return null; //  component does not render anything
}
