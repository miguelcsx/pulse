import type { AuthTokens, RegisterRequest, User } from "@pulse/drift/types";
import client from "./client";

export async function login(
  email: string,
  password: string,
): Promise<AuthTokens> {
  const response = await client.post<AuthTokens>("/auth/login", {
    email,
    password,
  });
  return response.data;
}

export async function register(data: RegisterRequest): Promise<AuthTokens> {
  const response = await client.post<AuthTokens>("/auth/register", data);
  return response.data;
}

export async function refreshToken(): Promise<{ access_token: string }> {
  const response = await client.post<{
    access_token: string;
  }>("/auth/refresh", {});
  return response.data;
}

export async function logout(): Promise<void> {
  await client.post("/auth/logout", {});
}

export async function getMe(): Promise<User> {
  const response = await client.get<User>("/me");
  return response.data;
}

export async function updateMe(
  data: Partial<Pick<User, "display_name" | "bio" | "avatar_url" | "location">>,
): Promise<User> {
  const response = await client.put<User>("/me", data);
  return response.data;
}
