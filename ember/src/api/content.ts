import type {
  Content,
  ContentFeedResponse,
  ContentType,
  FeedResponse,
  MediaAsset,
  ReactionKind,
} from "@pulse/drift/types";
import { FEED_PAGE_SIZE } from "@pulse/drift/constants";
import client from "./client";

export async function uploadContent(
  contentType: ContentType,
  file: File | null,
  body: string,
  tags: string[],
  mediaAssetId?: string,
): Promise<Content> {
  const formData = new FormData();
  formData.append("content_type", contentType);
  formData.append("body", body);
  if (mediaAssetId) {
    formData.append("media_asset_id", mediaAssetId);
  }
  if (file) {
    formData.append("file", file);
  }
  for (const tag of tags) {
    formData.append("tags", tag);
  }

  const response = await client.post<Content>("/content", formData, {
    headers: { "Content-Type": "multipart/form-data" },
  });
  return response.data;
}

export async function initMediaUpload(
  contentType: Exclude<ContentType, "text">,
  file: File,
): Promise<{ asset: MediaAsset; upload: { method: "PUT"; url: string } }> {
  const response = await client.post<{ asset: MediaAsset; upload: { method: "PUT"; url: string } }>(
    "/media/uploads/init",
    {
      content_type: contentType,
      filename: file.name,
      mime_type: file.type,
      size_bytes: file.size,
    },
  );
  return response.data;
}

export async function uploadMediaBinary(
  uploadURL: string,
  file: File,
  onProgress?: (percent: number) => void,
): Promise<MediaAsset> {
  const response = await client.put<MediaAsset>(uploadURL, file, {
    headers: { "Content-Type": file.type || "application/octet-stream" },
    onUploadProgress: (event) => {
      if (!onProgress || !event.total) return;
      const percent = Math.min(100, Math.round((event.loaded / event.total) * 100));
      onProgress(percent);
    },
  });
  return response.data;
}

export async function getMediaAsset(id: string): Promise<MediaAsset> {
  const response = await client.get<MediaAsset>(`/media/assets/${id}`);
  return response.data;
}

export async function getContent(id: string): Promise<Content> {
  const response = await client.get<Content>(`/content/${id}`);
  return response.data;
}

export async function deleteContent(id: string): Promise<void> {
  await client.delete(`/content/${id}`);
}

export async function getFeed(
  cursor?: string,
  limit: number = FEED_PAGE_SIZE,
): Promise<FeedResponse> {
  const params: Record<string, string | number> = { limit };
  if (cursor) {
    params.cursor = cursor;
  }
  const response = await client.get<FeedResponse>("/feed", { params });
  return response.data;
}

export async function getUserContent(
  userId: string,
  cursor?: string,
  limit: number = FEED_PAGE_SIZE,
): Promise<ContentFeedResponse> {
  const params: Record<string, string | number> = { limit };
  if (cursor) {
    params.cursor = cursor;
  }
  const response = await client.get<ContentFeedResponse>(
    `/users/${userId}/content`,
    { params },
  );
  return response.data;
}

export async function reactToContent(
  contentId: string,
  kind: ReactionKind,
): Promise<void> {
  await client.post(`/content/${contentId}/react`, { kind });
}

export async function removeReaction(
  contentId: string,
  kind: ReactionKind,
): Promise<void> {
  await client.delete(`/content/${contentId}/react`, { params: { kind } });
}
