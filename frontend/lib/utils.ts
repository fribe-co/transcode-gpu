import { type ClassValue, clsx } from "clsx"
import { twMerge } from "tailwind-merge"

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

export function formatBytes(bytes: number, decimals = 2): string {
  if (bytes === 0) return "0 Bytes";

  const k = 1024;
  const dm = decimals < 0 ? 0 : decimals;
  const sizes = ["Bytes", "KB", "MB", "GB", "TB"];

  const i = Math.floor(Math.log(bytes) / Math.log(k));

  return parseFloat((bytes / Math.pow(k, i)).toFixed(dm)) + " " + sizes[i];
}

export function formatDuration(seconds: number): string {
  const hours = Math.floor(seconds / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);
  const secs = Math.floor(seconds % 60);

  if (hours > 0) {
    return `${hours}h ${minutes}m ${secs}s`;
  }
  if (minutes > 0) {
    return `${minutes}m ${secs}s`;
  }
  return `${secs}s`;
}

export function formatBitrate(kbps: number): string {
  if (kbps >= 1000) {
    return `${(kbps / 1000).toFixed(1)} Mbps`;
  }
  return `${kbps} Kbps`;
}

export function getStatusColor(status: string): string {
  switch (status) {
    case "running":
      return "text-success";
    case "stopped":
      return "text-muted-foreground";
    case "error":
      return "text-destructive";
    case "starting":
    case "stopping":
      return "text-warning";
    default:
      return "text-muted-foreground";
  }
}

export function getStatusDotClass(status: string): string {
  switch (status) {
    case "running":
      return "status-dot-running";
    case "stopped":
      return "status-dot-stopped";
    case "error":
      return "status-dot-error";
    case "starting":
    case "stopping":
      return "status-dot-starting";
    default:
      return "status-dot-stopped";
  }
}

export function getStatusText(status: string): string {
  switch (status) {
    case "running":
      return "Çalışıyor";
    case "stopped":
      return "Durduruldu";
    case "error":
      return "Hata";
    case "starting":
      return "Başlatılıyor";
    case "stopping":
      return "Durduruluyor";
    default:
      return status;
  }
}

