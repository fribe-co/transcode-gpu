// Client-side: use empty string to make relative requests (handled by Next.js rewrites)
// Server-side: use backend container hostname
const API_BASE = typeof window !== 'undefined' 
  ? ""  // Client-side: relative path, Next.js rewrites to backend
  : (process.env.INTERNAL_API_URL || "http://backend:8080");

interface ApiResponse<T> {
  data?: T;
  error?: string;
}

class ApiClient {
  private getHeaders(): HeadersInit {
    const headers: HeadersInit = {
      "Content-Type": "application/json",
    };

    if (typeof window !== "undefined") {
      const token = localStorage.getItem("access_token");
      if (token) {
        headers["Authorization"] = `Bearer ${token}`;
      }
    }

    return headers;
  }

  private async request<T>(
    method: string,
    path: string,
    body?: unknown
  ): Promise<ApiResponse<T>> {
    try {
      const response = await fetch(`${API_BASE}${path}`, {
        method,
        headers: this.getHeaders(),
        body: body ? JSON.stringify(body) : undefined,
      });

      // Handle 204 No Content (empty response body)
      if (response.status === 204) {
        if (!response.ok) {
          return { error: "Bir hata oluştu" };
        }
        return { data: undefined as T };
      }

      // Try to parse JSON, but handle empty responses gracefully
      let data: any;
      const text = await response.text();
      if (text.trim() === "") {
        // Empty response body
        if (!response.ok) {
          return { error: "Bir hata oluştu" };
        }
        return { data: undefined as T };
      }

      try {
        data = JSON.parse(text);
      } catch (parseError) {
        // If JSON parse fails but status is OK, treat as success
        if (response.ok) {
          return { data: undefined as T };
        }
        return { error: "Geçersiz yanıt formatı" };
      }

      if (!response.ok) {
        return { error: data.error || "Bir hata oluştu" };
      }

      // Return data field if exists, otherwise return the whole response
      return { data: data.data !== undefined ? data.data : data };
    } catch (error) {
      return { error: "Ağ hatası" };
    }
  }

  // Auth
  async login(email: string, password: string) {
    const result = await this.request<{
      access_token: string;
      refresh_token: string;
      expires_at: string;
    }>("POST", "/api/v1/auth/login", { email, password });

    if (result.data) {
      localStorage.setItem("access_token", result.data.access_token);
      localStorage.setItem("refresh_token", result.data.refresh_token);
    }

    return result;
  }

  async logout() {
    localStorage.removeItem("access_token");
    localStorage.removeItem("refresh_token");
    return this.request("POST", "/api/v1/auth/logout");
  }

  async getMe() {
    return this.request<User>("GET", "/api/v1/auth/me");
  }

  // Channels
  async getChannels() {
    return this.request<Channel[]>("GET", "/api/v1/channels");
  }

  async getChannel(id: string) {
    return this.request<Channel>("GET", `/api/v1/channels/${id}`);
  }

  async createChannel(data: CreateChannelRequest) {
    return this.request<Channel>("POST", "/api/v1/channels", data);
  }

  async updateChannel(id: string, data: UpdateChannelRequest) {
    return this.request<Channel>("PUT", `/api/v1/channels/${id}`, data);
  }

  async deleteChannel(id: string) {
    return this.request<void>("DELETE", `/api/v1/channels/${id}`);
  }

  async startChannel(id: string) {
    return this.request<void>("POST", `/api/v1/channels/${id}/start`);
  }

  async stopChannel(id: string) {
    return this.request<void>("POST", `/api/v1/channels/${id}/stop`);
  }

  async restartChannel(id: string) {
    return this.request<void>("POST", `/api/v1/channels/${id}/restart`);
  }

  // Batch operations
  async batchStartChannels(ids: string[]) {
    return this.request<BatchResult>("POST", "/api/v1/channels/batch/start", {
      channel_ids: ids,
    });
  }

  async batchStopChannels(ids: string[]) {
    return this.request<BatchResult>("POST", "/api/v1/channels/batch/stop", {
      channel_ids: ids,
    });
  }

  async batchRestartChannels(ids: string[]) {
    return this.request<BatchResult>("POST", "/api/v1/channels/batch/restart", {
      channel_ids: ids,
    });
  }

  async batchDeleteChannels(ids: string[]) {
    return this.request<BatchResult>("POST", "/api/v1/channels/batch/delete", {
      channel_ids: ids,
    });
  }

  async getChannelMetrics(id: string) {
    return this.request<ProcessMetrics>("GET", `/api/v1/channels/${id}/metrics`);
  }

  async getAllChannelMetrics() {
    return this.request<ProcessMetrics[]>("GET", "/api/v1/channels/metrics");
  }

  async getChannelLogs(id: string) {
    return this.request<string[]>("GET", `/api/v1/channels/${id}/logs`);
  }

  // Logo upload
  async uploadLogo(file: File): Promise<ApiResponse<UploadLogoResponse>> {
    try {
      const formData = new FormData();
      formData.append("logo", file);

      const headers: HeadersInit = {};
      if (typeof window !== "undefined") {
        const token = localStorage.getItem("access_token");
        if (token) {
          headers["Authorization"] = `Bearer ${token}`;
        }
      }

      const response = await fetch(`${API_BASE}/api/v1/uploads/logo`, {
        method: "POST",
        headers,
        body: formData,
      });

      const data = await response.json();

      if (!response.ok) {
        return { error: data.error || "Logo yüklenemedi" };
      }

      return { data: data.data };
    } catch (error) {
      return { error: "Ağ hatası" };
    }
  }

  async deleteLogo(filename: string) {
    return this.request<void>("DELETE", `/api/v1/uploads/logo/${filename}`);
  }

  // Settings
  async getSettings() {
    return this.request<Settings>("GET", "/api/v1/settings");
  }

  async updateSettings(data: UpdateSettingsRequest) {
    return this.request<Settings>("PUT", "/api/v1/settings", data);
  }
  // System info
  async getSystemInfo() {
    return this.request<SystemInfo>("GET", "/api/v1/system/info");
  }
}

export const api = new ApiClient();

// Types
export interface User {
  id: string;
  email: string;
  name: string;
  role: "admin" | "operator" | "viewer";
}

export interface LogoConfig {
  path: string;
  x: number;
  y: number;
  width: number;
  height: number;
  opacity: number;
}

export interface OutputConfig {
  codec: string;
  bitrate: string;
  resolution: string;
  preset: string;
  profile: string;
}

export interface Channel {
  id: string;
  name: string;
  source_url: string;
  output_url?: string;
  logo?: LogoConfig;
  output_config?: OutputConfig;
  status: "stopped" | "starting" | "running" | "error" | "stopping";
  auto_restart: boolean;
  created_at: string;
  updated_at: string;
}

export interface ProcessMetrics {
  channel_id: string;
  pid: number;
  started_at: string;
  cpu_usage: number;
  memory_usage: number;
  input_bitrate: number;
  output_bitrate: number;
  dropped_frames: number;
  fps: number;
  speed: number;
  uptime: number;
}

export interface CreateChannelRequest {
  name: string;
  source_url: string;
  logo?: LogoConfig;
  output_config?: OutputConfig;
}

export interface UpdateChannelRequest {
  name?: string;
  source_url?: string;
  logo?: LogoConfig;
  output_config?: OutputConfig;
}

export interface UploadLogoResponse {
  path: string;
  filename: string;
  url: string;
}

export interface Settings {
  max_channels: number;
  segment_time: number;
  playlist_size: number;
  log_retention: number;
  default_preset: string;
  default_bitrate: string;
  default_resolution: string;
  default_profile: string;
}

export interface UpdateSettingsRequest {
  max_channels?: number;
  segment_time?: number;
  playlist_size?: number;
  log_retention?: number;
  default_preset?: string;
  default_bitrate?: string;
  default_resolution?: string;
  default_profile?: string;
}

export interface BatchResult {
  success: string[];
  failed: Array<{
    channel_id: string;
    error: string;
  }>;
}

export interface GPUInfo {
  id: string;
  name: string;
  utilization: number;
  memory_used: number;
  memory_total: number;
  temperature: number;
}

export interface SystemInfo {
  cpu_cores: number;
  cpu_threads: number;
  cpu_usage: number;
  memory_total: number;
  memory_used: number;
  memory_available: number;
  memory_percent: number;
  load_average_1: number;
  load_average_5: number;
  load_average_15: number;
  uptime: number;
  gpus?: GPUInfo[];
}
