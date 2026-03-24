import { request } from "./request.js";

export function requestLLMProvider(payload) {
  return request.post("/api/v1/chat", payload);
}
