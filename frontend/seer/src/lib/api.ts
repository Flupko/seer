import { BalanceSchema, CategorySchema, ChangePasswordPayload, Currency, LoginFormValues, MarketSearch, MarketViewSchema, MetadataSchema, PlaceBet, ProfileCompletionFormValues, RegisterFormValues, SessionsSchema, SetPasswordPayload, UpdateUserPreferences, User, UserPreferencesSchema, UserSchema } from "@/lib/definitions";
import z from "zod";

export const API_BASE_URL = process.env.NEXT_PUBLIC_API_BASE_URL || ""

export interface APIError {
  message: string;
  errors?: Array<{ field: string; message: string }>;
}

export const register = async (formData: RegisterFormValues) => {
  const response = await fetch(`${API_BASE_URL}/auth/register`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    credentials: "include",
    body: JSON.stringify(formData),
  });

  const data = await response.json();

  if (!response.ok) {

    const error: APIError = {
      message: data.message,
      errors: data.errors,
    };
    throw error;
  }

  return null;
};

export const login = async (formData: LoginFormValues) => {

  const response = await fetch(`${API_BASE_URL}/auth/login`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    credentials: "include",
    body: JSON.stringify(formData),
  });

  const data = await response.json();

  if (!response.ok) {
    const error: APIError = {
      message: data.message,
      errors: data.errors,
    };
    throw error;
  }

  return null;

}

const UserResponseSchema = z.union([UserSchema, z.null()]);

export const getUser = async (cookie?: string): Promise<User | null> => {

  const headers: HeadersInit = {};

  // if on server (prefetch) and cookie is passed, forward it
  if (cookie) {
    headers["Cookie"] = cookie;
  }

  const response = await fetch(`${API_BASE_URL}/user/me`, {
    credentials: "include",
    headers,
  });

  const data = await response.json();

  if (!response.ok) {
    if (response.status === 401) {
      return null;
    }

    const error: APIError = {
      message: data.message || "Failed to fetch user",
    };
    throw error;
  }

  // Validate response
  const result = UserResponseSchema.safeParse(data);

  if (!result.success) {
    throw new Error("Invalid user data");
  }

  return result.data;
};

export const completeProfile = async (formData: ProfileCompletionFormValues) => {

  const response = await fetch(`${API_BASE_URL}/auth/complete-profile`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    credentials: "include",
    body: JSON.stringify(formData),
  });

  const data = await response.json();

  if (!response.ok) {
    const error: APIError = {
      message: data.message,
      errors: data.errors,
    };
    throw error;
  }

  return null;
}

export const logout = async () => {
  const response = await fetch(`${API_BASE_URL}/auth/logout`, {
    method: "POST",
    credentials: "include", // Important: sends cookies
  });

  if (!response.ok) {
    const data = await response.json();
    const error: APIError = {
      message: data.message || "Logout failed",
    };
    throw error;
  }

  return response.json();
};


export const getSessions = async (showInactive: boolean) => {

  // If showInactive is false, we add a query param to only fetch active sessions
  let url = `${API_BASE_URL}/auth/sessions`;
  if (showInactive) {
    url += "?showInactive=true";
  }

  const response = await fetch(url, {
    credentials: "include",
  });

  const data = await response.json();
  if (!response.ok) {
    const error: APIError = {
      message: data.message || "Failed to fetch sessions",
    };
    throw error;
  }


  const result = SessionsSchema.safeParse(data);
  if (!result.success) {
    throw new Error("Invalid session data");
  }

  return result.data;

}

export const revokeSession = async (sessionId: string) => {
  const response = await fetch(`${API_BASE_URL}/auth/sessions/${sessionId}/revoke`, {
    method: "PATCH",
    credentials: "include",
  });

  if (!response.ok) {
    const data = await response.json();
    const error: APIError = {
      message: data.message || "Failed to revoke session",
    };
    throw error;
  }

  return null;
}

export const changePassword = async (formData: ChangePasswordPayload) => {

  const response = await fetch(`${API_BASE_URL}/auth/password/change`, {
    method: "PATCH",
    headers: {
      "Content-Type": "application/json",
    },
    credentials: "include",
    body: JSON.stringify(formData),
  });

  const data = await response.json();

  if (!response.ok) {

    const error: APIError = {
      message: data.message,
      errors: data.errors,
    };
    throw error;
  }

  return null;
};

export const setPassword = async (formData: SetPasswordPayload) => {

  const response = await fetch(`${API_BASE_URL}/auth/password/set`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    credentials: "include",
    body: JSON.stringify(formData),
  });

  const data = await response.json();

  if (!response.ok) {

    const error: APIError = {
      message: data.message,
      errors: data.errors,
    };
    throw error;
  }

  return null;
};

export const getUserPreferences = async () => {

  const response = await fetch(`${API_BASE_URL}/user/prefs`, {
    credentials: "include",
  });

  const data = await response.json();

  if (!response.ok) {
    const error: APIError = {
      message: data.message || "Failed to fetch user preferences",
      errors: data.errors,
    };
    throw error;
  }

  const result = UserPreferencesSchema.safeParse(data);
  if (!result.success) {
    throw new Error("Invalid preferences data");
  }

  return result.data;
}

export const updateUserPreferences = async (preferences: UpdateUserPreferences) => {
  const response = await fetch(`${API_BASE_URL}/user/prefs`, {
    method: "PATCH",
    headers: {
      "Content-Type": "application/json",
    },
    credentials: "include",
    body: JSON.stringify(preferences),
  });

  const data = await response.json();
  if (!response.ok) {
    const error: APIError = {
      message: data.message || "Failed to update preferences",
      errors: data.errors,
    };
    throw error;
  }

  return null;
}

const MarketSearchResSchema = z.object({
  markets: MarketViewSchema.array(),
  metadata: MetadataSchema,
});

export type MarketSearchRes = z.infer<typeof MarketSearchResSchema>

export const searchMarket = async (search: MarketSearch) => {
  const params = new URLSearchParams();
  if (search.query) params.append("query", search.query);
  if (search.categorySlug) params.append("categorySlug", search.categorySlug);
  params.append("sort", search.sort);
  params.append("status", search.status);
  params.append("pageSize", search.pageSize.toString());
  params.append("page", search.page.toString());
  if (search.status) {
    params.append("status", search.status);
  }

  params.append("page", search.page.toString());

  const response = await fetch(`${API_BASE_URL}/market/search?${params.toString()}`, {
    credentials: "include",
  });

  const data = await response.json();

  if (!response.ok) {
    const error: APIError = {
      message: data.message || "Failed to search markets",
      errors: data.errors,
    };
    throw error;
  }

  // Safe parse
  const result = MarketSearchResSchema.safeParse(data);
  if (!result.success) {
    throw new Error("Invalid market search data");
  }

  return result.data;

}

const CategoriesResponseSchema = z.array(CategorySchema)

export const getFeaturedCategories = async () => {

  const response = await fetch(`${API_BASE_URL}/market/categories/featured`, {
    cache: "force-cache",
  })
  const data = await response.json();

  if (!response.ok) {
    const error: APIError = {
      message: data.message || "Failed to fetch categories",
      errors: data.errors,
    };
    throw error;
  }

  const result = CategoriesResponseSchema.safeParse(data);
  if (!result.success) {
    throw new Error("Invalid categories data", result.error);
  }

  return result.data;
}

export const getMarket = async (marketId: string) => {
  const response = await fetch(`${API_BASE_URL}/market/search/${marketId}`, {});

  const data = await response.json();

  if (!response.ok) {
    const error: APIError = {
      message: data.message || "Failed to fetch market",
      errors: data.errors,
    };
    throw error;
  }

  const result = MarketViewSchema.safeParse(data);
  if (!result.success) {
    throw new Error("Invalid market data");
  }

  return result.data;
}


export const getBalance = async (currency: Currency) => {
  const response = await fetch(`${API_BASE_URL}/user/balance/${currency}`, {
    credentials: "include",
  });

  const data = await response.json();
  if (!response.ok) {
    const error: APIError = {
      message: data.message || "Failed to fetch balance",
      errors: data.errors,
    };
    throw error;
  }

  const result = BalanceSchema.safeParse(data);
  if (!result.success) {
    throw new Error("Invalid balance data");
  }

  return result.data;

}

export const postBet = async (placeBet: PlaceBet) => {
  const response = await fetch(`${API_BASE_URL}/market/bet`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    credentials: "include",
    body: JSON.stringify(placeBet),
  });

  const data = await response.json();

  if (!response.ok) {
    const error: APIError = {
      message: data.message || "Failed to place bet",
      errors: data.errors,
    };

    throw error;
  }

  return null;
}