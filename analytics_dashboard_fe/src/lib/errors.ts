/** Error carrying the backend's stable error code, so the UI maps code → message. */
export class ApiError extends Error {
  code: string;
  status: number;
  constructor(code: string, status: number, message?: string) {
    super(message ?? code);
    this.code = code;
    this.status = status;
    this.name = "ApiError";
  }
}

/** Code → user-facing message. The backend never sends localized text; the UI
 *  owns the wording. */
export const ERROR_MESSAGES: Record<string, string> = {
  AUTH_INVALID_CREDENTIALS: "Incorrect email or password.",
  AUTH_TOKEN_MISSING: "Please sign in to continue.",
  AUTH_TOKEN_INVALID: "Your session is invalid. Please sign in again.",
  AUTH_TOKEN_EXPIRED: "Your session has expired. Please sign in again.",
  AUTH_FORBIDDEN: "You don't have permission to do that.",
  VALIDATION_ERROR: "Please check your input and try again.",
  RATE_LIMITED: "You're sending questions too quickly. Please wait a moment and try again.",
  PAYLOAD_TOO_LARGE: "That request is too large. Please shorten it and try again.",
  INTERNAL_ERROR: "Something went wrong on our end. Please try again.",
  NETWORK_ERROR: "Couldn't reach the server. Check your connection and try again.",
};

export function messageForCode(code: string): string {
  return ERROR_MESSAGES[code] ?? "An unexpected error occurred.";
}

/** Extracts a friendly message from any thrown value. */
export function messageForError(err: unknown): string {
  if (err instanceof ApiError) return messageForCode(err.code);
  return messageForCode("NETWORK_ERROR");
}
