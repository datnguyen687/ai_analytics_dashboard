"use client";

import { createContext, useContext, useEffect, useState, type ReactNode } from "react";
import { getMeta, type MetaResponse } from "./api";

interface MetaState {
  meta: MetaResponse | null;
  loading: boolean;
  error: string | null;
}

const MetaContext = createContext<MetaState>({ meta: null, loading: true, error: null });

/** Fetches the filter metadata (dimension values + date bounds) once and shares
 *  it with every page, so the filter controls no longer need a bundled dataset. */
export function MetaProvider({ children }: { children: ReactNode }) {
  const [state, setState] = useState<MetaState>({ meta: null, loading: true, error: null });

  useEffect(() => {
    const ctrl = new AbortController();
    getMeta(ctrl.signal)
      .then((meta) => setState({ meta, loading: false, error: null }))
      .catch((e: unknown) => {
        if ((e as Error).name === "AbortError") return;
        setState({ meta: null, loading: false, error: (e as Error).message });
      });
    return () => ctrl.abort();
  }, []);

  return <MetaContext.Provider value={state}>{children}</MetaContext.Provider>;
}

export function useMeta() {
  return useContext(MetaContext);
}
