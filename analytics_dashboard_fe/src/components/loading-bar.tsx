"use client";

import { useEffect, useState } from "react";
import { onLoadingChange } from "@/lib/api";

/** Thin indeterminate progress bar pinned to the very top of the viewport,
 *  shown whenever any API request is in flight — so a slow call never reads as
 *  a frozen page. */
export function LoadingBar() {
  const [active, setActive] = useState(false);
  useEffect(() => onLoadingChange(setActive), []);

  return (
    <div
      aria-hidden
      className={`pointer-events-none fixed inset-x-0 top-0 z-[60] h-0.5 overflow-hidden transition-opacity duration-300 ${
        active ? "opacity-100" : "opacity-0"
      }`}
    >
      <div className="loading-bar-track h-full w-full" />
    </div>
  );
}
