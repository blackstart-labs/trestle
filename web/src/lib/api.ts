/** Same-origin JSON requests through the existing session and CSRF boundary. */
export async function api(path: string, method = "GET", body?: unknown) {
  const csrf =
    document.cookie
      .split(";")
      .map((x) => x.trim())
      .find((x) => x.startsWith("trestle-csrf="))
      ?.slice(13) || "";
  let response: Response;
  try {
    response = await fetch(path, {
      method,
      cache: "no-store",
      headers: {
        Accept: "application/json",
        ...(method !== "GET" ? { "X-CSRF-Token": csrf } : {}),
        ...(body !== undefined ? { "Content-Type": "application/json" } : {}),
      },
      ...(body !== undefined ? { body: JSON.stringify(body) } : {}),
    });
  } catch {
    throw new Error(
      "Could not reach the server. Your draft is unchanged. Check your connection and retry.",
    );
  }
  if (response.status === 204) return null;
  const data = await response.json().catch(() => null);
  if (!response.ok || !data)
    throw new Error(
      data?.error?.message ||
        "The request could not be completed. Check your access and retry.",
    );
  return data;
}
export function message(error: unknown) {
  return error instanceof Error
    ? error.message
    : "The request failed. Please retry.";
}
