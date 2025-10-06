"use client";

import { User } from "@/lib/definitions";
import { useQuery } from "@tanstack/react-query";
import React, { createContext, useContext } from "react";
import * as api from "@/lib/api";
import { toast } from "react-toastify";

interface AuthContext {
  user: User | null;
  isLoading: boolean;
}

const AuthContext = createContext<AuthContext | null>(null);

export default function AuthProvider({ children }: React.PropsWithChildren) {
  const { data: user, isLoading, isError } = useQuery({
    queryKey: ["user"],
    queryFn: api.getUser,
    retry: false,
    staleTime: 60 * 60 * 1000 // 1 hour
  });

  return (
    <AuthContext.Provider value={{ user: user ?? null, isLoading }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useSession() {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error("useSession must be used within AuthProvider");
  }
  return context;
}
