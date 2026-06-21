import { useEffect, useState } from "react";
import { api, getToken } from "../api/client";

export function PersonPhoto({
  photoId,
  alt = "",
  className,
  fallback = null,
}: {
  photoId: string;
  alt?: string;
  className?: string;
  fallback?: React.ReactNode;
}) {
  const [src, setSrc] = useState<string | null>(null);
  const [failed, setFailed] = useState(false);

  useEffect(() => {
    let objectUrl: string | null = null;
    let cancelled = false;
    setFailed(false);
    setSrc(null);

    const token = getToken();
    void fetch(api.photoUrl(photoId), {
      headers: token ? { Authorization: `Bearer ${token}` } : {},
    })
      .then((res) => {
        if (!res.ok) throw new Error("photo fetch failed");
        return res.blob();
      })
      .then((blob) => {
        if (cancelled) return;
        objectUrl = URL.createObjectURL(blob);
        setSrc(objectUrl);
      })
      .catch(() => {
        if (!cancelled) setFailed(true);
      });

    return () => {
      cancelled = true;
      if (objectUrl) URL.revokeObjectURL(objectUrl);
    };
  }, [photoId]);

  if (failed || !src) return <>{fallback}</>;
  return <img src={src} alt={alt} className={className} />;
}