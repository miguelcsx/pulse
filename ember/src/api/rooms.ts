import type { Room } from "@pulse/drift/types";
import client from "./client";

interface RoomMembershipResponse {
  member_count: number;
}

export async function getRooms(): Promise<Room[]> {
  const response = await client.get<Room[]>("/rooms");
  return response.data;
}

export async function enterRoom(id: string): Promise<number> {
  const response = await client.post<RoomMembershipResponse>(`/rooms/${id}/enter`);
  return response.data.member_count;
}

export async function leaveRoom(id: string): Promise<number> {
  const response = await client.post<RoomMembershipResponse>(`/rooms/${id}/leave`);
  return response.data.member_count;
}
