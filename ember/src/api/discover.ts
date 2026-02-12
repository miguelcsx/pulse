import type { DiscoverResponse } from "@pulse/drift/types";
import client from "./client";

export async function getDiscover(): Promise<DiscoverResponse> {
  const response = await client.get<DiscoverResponse>("/discover");
  return response.data;
}
