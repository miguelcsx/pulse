import type { Suggestion, UserProfile } from "@pulse/drift/types";
import client from "./client";

export async function getSuggestions(limit?: number): Promise<Suggestion[]> {
  const response = await client.get<Suggestion[]>("/suggestions", {
    params: limit ? { limit } : undefined,
  });
  return response.data;
}

export async function followUser(id: string): Promise<void> {
  await client.post(`/follow/${id}`);
}

export async function unfollowUser(id: string): Promise<void> {
  await client.delete(`/follow/${id}`);
}

export async function blockUser(id: string): Promise<void> {
  await client.post(`/block/${id}`);
}

export async function unblockUser(id: string): Promise<void> {
  await client.delete(`/block/${id}`);
}

export async function getUserProfile(id: string): Promise<UserProfile> {
  const response = await client.get<UserProfile>(`/users/${id}`);
  return response.data;
}
