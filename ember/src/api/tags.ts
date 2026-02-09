import type { Tag } from "@pulse/drift/types";
import client from "./client";

export async function getTags(): Promise<Tag[]> {
  const response = await client.get<Tag[]>("/tags");
  return response.data;
}

export async function searchTags(query: string): Promise<Tag[]> {
  const response = await client.get<Tag[]>("/tags/search", { params: { q: query } });
  return response.data;
}
