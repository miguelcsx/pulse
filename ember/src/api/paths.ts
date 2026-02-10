import type { PaginatedResponse, Path, PathCreate } from "@pulse/drift/types";
import client from "./client";

export async function getPaths(
  cursor?: string,
  limit: number = 20,
): Promise<PaginatedResponse<Path>> {
  const params: Record<string, string | number> = { limit };
  if (cursor) {
    params.cursor = cursor;
  }

  const response = await client.get<PaginatedResponse<Path>>("/paths", {
    params,
  });
  return response.data;
}

export async function getPath(id: string): Promise<Path> {
  const response = await client.get<Path>(`/paths/${id}`);
  return response.data;
}

export async function createPath(data: PathCreate): Promise<Path> {
  const response = await client.post<Path>("/paths", data);
  return response.data;
}

export async function followPath(id: string): Promise<void> {
  await client.post(`/paths/${id}/follow`);
}

export async function unfollowPath(id: string): Promise<void> {
  await client.delete(`/paths/${id}/follow`);
}
