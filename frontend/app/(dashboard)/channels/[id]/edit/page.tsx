"use client";

import { useEffect, useState, useRef, useCallback } from "react";
import { useParams, useRouter } from "next/navigation";
import { motion } from "framer-motion";
import {
  ArrowLeft,
  Upload,
  Save,
  Loader2,
  Image as ImageIcon,
  Trash2,
  Move,
  Eye,
  Link as LinkIcon,
  Copy,
  Check,
  Play,
  ExternalLink,
  Terminal,
  RefreshCw,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { api, Channel, LogoConfig } from "@/lib/api";
import { useToast } from "@/hooks/use-toast";
import { cn } from "@/lib/utils";

const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";
const CDN_BASE = "https://cdn.cashbacktv.live";
const getFullUrl = (path: string) => {
  // If path is already a full URL (starts with http), return as is
  if (path.startsWith("http://") || path.startsWith("https://")) {
    return path;
  }
  // If path starts with /streams/, use CDN
  if (path.startsWith("/streams/")) {
    return `${CDN_BASE}${path}`;
  }
  // Otherwise use current origin
  if (typeof window !== 'undefined') {
    return `${window.location.origin}${path}`;
  }
  return path;
};

export default function ChannelEditPage() {
  const params = useParams();
  const router = useRouter();
  const { toast } = useToast();
  const channelId = params.id as string;

  const [channel, setChannel] = useState<Channel | null>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [uploading, setUploading] = useState(false);
  const [copied, setCopied] = useState(false);
  const [logs, setLogs] = useState<string[]>([]);
  const [loadingLogs, setLoadingLogs] = useState(false);
  const [showLogs, setShowLogs] = useState(false);
  const logsEndRef = useRef<HTMLDivElement>(null);
  const videoRef = useRef<HTMLVideoElement>(null);

  // Form state
  const [name, setName] = useState("");
  const [sourceUrl, setSourceUrl] = useState("");
  const [bitrate, setBitrate] = useState("5000k");
  const [preset, setPreset] = useState("veryfast");
  
  // Logo state
  const [logoUrl, setLogoUrl] = useState<string | null>(null);
  const [logoFilename, setLogoFilename] = useState<string | null>(null);
  const [logoPath, setLogoPath] = useState<string | null>(null);
  const [logoX, setLogoX] = useState(10);
  const [logoY, setLogoY] = useState(10);
  const [logoWidth, setLogoWidth] = useState(200);
  const [logoHeight, setLogoHeight] = useState(100);
  const [logoOpacity, setLogoOpacity] = useState(1.0);

  // Drag state
  const [isDragging, setIsDragging] = useState(false);
  const [dragStart, setDragStart] = useState({ x: 0, y: 0 });
  const previewRef = useRef<HTMLDivElement>(null);

  // Video dimensions - will be set from source video metadata
  const [videoWidth, setVideoWidth] = useState(1920);
  const [videoHeight, setVideoHeight] = useState(1080);

  useEffect(() => {
    const fetchChannel = async () => {
      const result = await api.getChannel(channelId);
      if (result.error) {
        toast({ title: "Hata", description: result.error, variant: "destructive" });
        router.push("/channels");
        return;
      }

      if (result.data) {
        const ch = result.data;
        setChannel(ch);
        setName(ch.name);
        setSourceUrl(ch.source_url);
        setBitrate(ch.output_config?.bitrate || "4000k");
        setPreset(ch.output_config?.preset || "veryfast");

        // Load resolution from channel settings if available
        if (ch.output_config?.resolution) {
          const resolutionParts = ch.output_config.resolution.split("x");
          if (resolutionParts.length === 2) {
            const width = parseInt(resolutionParts[0], 10);
            const height = parseInt(resolutionParts[1], 10);
            if (width > 0 && height > 0) {
              setVideoWidth(width);
              setVideoHeight(height);
            }
          }
        }

        if (ch.logo) {
          setLogoPath(ch.logo.path);
          // Use relative path for logo, handled by next.config.js rewrite
          const logoRelPath = ch.logo.path.startsWith("/") ? ch.logo.path : `/logos/${ch.logo.path}`;
          setLogoUrl(logoRelPath);
          setLogoX(ch.logo.x);
          setLogoY(ch.logo.y);
          setLogoWidth(ch.logo.width);
          setLogoHeight(ch.logo.height);
          setLogoOpacity(ch.logo.opacity);
        }
      }
      setLoading(false);
    };

    fetchChannel();
  }, [channelId, router, toast]);

  // Auto-scroll logs to bottom
  useEffect(() => {
    if (showLogs && logsEndRef.current) {
      logsEndRef.current.scrollIntoView({ behavior: "smooth" });
    }
  }, [logs, showLogs]);

  // Auto-refresh logs if channel is running
  useEffect(() => {
    if (!channel || channel.status !== "running" || !showLogs) return;

    const fetchLogs = async () => {
      const result = await api.getChannelLogs(channelId);
      if (result.data) {
        setLogs(result.data);
      }
    };

    fetchLogs();
    const interval = setInterval(fetchLogs, 2000); // Refresh every 2 seconds
    return () => clearInterval(interval);
  }, [channel, channelId, showLogs]);

  // Check if URL is a stream (m3u8, ts, mpegts, or other video formats)
  const isStreamUrl = (url: string): boolean => {
    if (!url) return false;
    const lowerUrl = url.toLowerCase();
    return (
      lowerUrl.includes('.m3u8') ||
      lowerUrl.includes('.m3u') ||
      lowerUrl.includes('/hls/') ||
      lowerUrl.includes('mpegurl') ||
      lowerUrl.includes('application/vnd.apple.mpegurl') ||
      lowerUrl.endsWith('.ts') ||
      lowerUrl.includes('.ts?') ||
      lowerUrl.includes('.ts&') ||
      lowerUrl.includes('/playlist.m3u8') ||
      lowerUrl.includes('/index.m3u8') ||
      lowerUrl.includes('mpegts') ||
      lowerUrl.includes('mpeg-ts') ||
      lowerUrl.includes('video/mp2t') ||
      lowerUrl.includes('application/x-mpegurl') ||
      lowerUrl.includes('transportstream') ||
      lowerUrl.includes('transport_stream')
    );
  };

  // Check if URL is a direct TS stream (not playlist)
  // This includes URLs without .ts extension that are likely TS streams
  const isDirectTSStream = (url: string): boolean => {
    if (!url) return false;
    const lowerUrl = url.toLowerCase();
    
    // Explicit .ts extension
    if ((lowerUrl.endsWith('.ts') || lowerUrl.includes('.ts?')) &&
        !lowerUrl.includes('.m3u8') &&
        !lowerUrl.includes('.m3u')) {
      return true;
    }
    
    // URLs without extension that might be TS streams
    // Common patterns: port numbers, numeric endings, short paths
    const urlObj = new URL(url);
    const pathname = urlObj.pathname.toLowerCase();
    
    // If URL has no file extension and looks like a stream endpoint
    if (!pathname.match(/\.(m3u8|m3u|mp4|webm|ogg|avi|mov)$/i)) {
      // Check if it's likely a stream (has port, short path, numeric ending)
      if (urlObj.port || 
          pathname.split('/').length <= 4 ||
          /\/\d+$/.test(pathname)) {
        return true;
      }
    }
    
    return false;
  };

  // Get preview URL - only show output stream when channel is running
  const getPreviewUrl = useCallback((): string | null => {
    // Only show preview when channel is running - use output stream
    if (channel?.status === "running" && channel?.output_url) {
      const path = channel.output_url || `/streams/${channelId}/index.m3u8`;
      return getFullUrl(path);
    }
    return null;
  }, [channel?.status, channel?.output_url, channelId]);

  // Setup video player for both source and output streams
  useEffect(() => {
    if (typeof window === "undefined") return;
    
    // Wait a bit for video ref to be available
    const checkVideoRef = () => {
      if (!videoRef.current) {
        console.log("Video ref not available, retrying...");
        setTimeout(checkVideoRef, 100);
        return;
      }

      const video = videoRef.current;
      const previewUrl = getPreviewUrl();
    
      console.log("Preview URL:", previewUrl);
      console.log("Source URL:", sourceUrl);
      console.log("Channel status:", channel?.status);
      console.log("Output URL:", channel?.output_url);
      
      if (!previewUrl) {
        console.log("No preview URL available");
        return;
      }

      let hls: any = null;
      let blobUrl: string | null = null;

      const initHls = async () => {
      try {
        console.log("Initializing video player for URL:", previewUrl);
        console.log("Is stream URL:", isStreamUrl(previewUrl));
        
        // Clear any existing source and HLS instance
        if (video.src) {
          video.pause();
          video.src = "";
          video.load();
        }
        
        // Always try HLS.js first for any URL (it can handle m3u8, ts, etc.)
        console.log("Loading HLS.js...");
        const HlsModule = await import("hls.js");
        const Hls = HlsModule.default || HlsModule;

        console.log("HLS.js loaded, supported:", Hls && typeof Hls.isSupported === "function" ? Hls.isSupported() : false);

        if (Hls && typeof Hls.isSupported === "function" && Hls.isSupported()) {
          console.log("Using HLS.js to play stream");
          
          // Destroy any existing HLS instance
          if (hls) {
            hls.destroy();
          }
          
          // Check if it's a direct TS stream (including URLs without .ts extension)
          const isDirectTS = isDirectTSStream(previewUrl);
          console.log("Is direct TS stream:", isDirectTS);
          console.log("Preview URL:", previewUrl);
          
          hls = new Hls({
            enableWorker: true,
            lowLatencyMode: false, // Disable low latency mode for better buffering
            backBufferLength: 180, // Keep more segments in back buffer (increase from 90 to 180)
            debug: false,
            // Optimize buffer settings to prevent stalling - prioritize smooth playback over low latency
            maxBufferLength: 180, // Allow larger buffer (180 seconds = 3 minutes) - increase from 120
            maxMaxBufferLength: 600, // Maximum buffer length
            maxBufferSize: 200 * 1000 * 1000, // 200MB buffer (increase from 120MB) - more aggressive buffering
            maxBufferHole: 1.0, // Increased tolerance for gaps in buffer (from 0.5 to 1.0)
            // Live sync settings for smooth playback - accept more latency for stability
            liveSyncDurationCount: 5, // Sync to live edge every 5 segments (increase from 3) - less frequent sync
            liveMaxLatencyDurationCount: 30, // Max latency: 30 segments (increase from 20) - allow more latency
            liveDurationInfinity: false, // Don't buffer infinite duration
            // Start loading immediately with more aggressive buffering
            autoStartLoad: true,
            startLevel: -1, // Auto-detect quality
            // Retry settings - more retries for better reliability
            manifestLoadingTimeOut: 15000, // 15 seconds timeout (increase from 10)
            manifestLoadingMaxRetry: 5, // More retries (increase from 3)
            manifestLoadingRetryDelay: 1000,
            levelLoadingTimeOut: 15000, // 15 seconds (increase from 10)
            levelLoadingMaxRetry: 5, // More retries (increase from 3)
            levelLoadingRetryDelay: 1000,
            fragLoadingTimeOut: 30000, // 30 seconds for segment loading (increase from 20)
            fragLoadingMaxRetry: 10, // More retries for segments (increase from 6)
            fragLoadingRetryDelay: 1000,
            // Don't abort on gaps - try to fill them - more tolerant
            abrEwmaFastLive: 3.0,
            abrEwmaSlowLive: 9.0,
            abrEwmaFastVoD: 3.0,
            abrEwmaSlowVoD: 9.0,
            // Buffer stalling recovery - more aggressive recovery
            highBufferWatchdogPeriod: 2, // Check buffer health every 2 seconds
            nudgeOffset: 0.1, // Nudge playback by 0.1s on stall
            nudgeMaxRetry: 10, // More retries (increase from 5)
            maxStarvationDelay: 8, // Max delay before considering starving (increase from 4)
            maxLoadingDelay: 8, // Increase from 4
            // XHR setup
            xhrSetup: (xhr, url) => {
              // Handle CORS if needed
              xhr.withCredentials = false;
              // Add longer timeout for segment requests
              xhr.timeout = 30000; // Increase from 20000 to 30000
              console.log("Loading segment:", url);
            },
          });
          
          // Set up error handlers before loading
          hls.on(Hls.Events.ERROR, (event: any, data: any) => {
            // Handle buffer stalling (non-fatal errors) - these are expected in live streams
            if (!data.fatal && data.type === "mediaError" && data.details === "bufferStalledError") {
              // Buffer stalling is expected in live streams - HLS.js will automatically recover
              // Only log if buffer is critically low (less than 0.1 seconds)
              if (data.buffer && data.buffer < 0.1) {
                console.warn("HLS buffer critically low:", data.buffer, "seconds");
              }
              return; // Let HLS.js handle buffer recovery automatically
            }
            
            // Handle other non-fatal errors silently (don't spam console)
            if (!data.fatal) {
              // Only log non-fatal errors that are not buffer-related
              if (data.type !== "mediaError" || 
                  (data.details !== "bufferStalledError" && 
                   data.details !== "bufferSeekingOver" && 
                   data.details !== "bufferSeeked")) {
                console.warn("HLS non-fatal error:", data.type, data.details);
              }
              return;
            }
            
            // Handle fatal errors
            console.error("HLS fatal error:", data);
            switch (data.type) {
              case Hls.ErrorTypes.NETWORK_ERROR:
                console.log("Network error, retrying...");
                setTimeout(() => {
                  hls.startLoad();
                }, 1000);
                break;
              case Hls.ErrorTypes.MEDIA_ERROR:
                console.log("Media error, recovering...");
                hls.recoverMediaError();
                break;
              default:
                console.error("Fatal error, trying fallback");
                hls.destroy();
                // Try native HLS as fallback
                if (video.canPlayType("application/vnd.apple.mpegurl")) {
                  console.log("Trying native HLS as fallback");
                  video.src = previewUrl;
                  video.load();
                  video.play().catch((err) => {
                    console.error("Native HLS play error:", err);
                  });
                }
                toast({
                  title: "Stream Hatası",
                  description: "Stream yüklenirken hata oluştu: " + (data.details || data.message || "Bilinmeyen hata"),
                  variant: "destructive",
                });
                break;
            }
          });
          
          hls.on(Hls.Events.MANIFEST_PARSED, () => {
            console.log("HLS manifest parsed, starting playback");
            video.play().catch((err) => {
              console.error("Video play error:", err);
              toast({
                title: "Oynatma Hatası",
                description: "Video oynatılamadı: " + err.message,
                variant: "destructive",
              });
            });
          });
          
          hls.on(Hls.Events.LEVEL_LOADED, () => {
            console.log("HLS level loaded");
          });
          
          // Load the source
          console.log("Loading HLS source:", previewUrl);
          
          // For direct TS streams (including URLs without .ts extension), try different approaches
          if (isDirectTSStream(previewUrl)) {
            console.log("Direct TS stream detected (video/mp2t), trying HLS.js with virtual playlist");
            
            // For MPEG-TS streams, we need to use HLS.js with a virtual playlist
            // Create a proper M3U8 playlist pointing to the TS stream
            // Use absolute URL to ensure proper loading
            const tsUrl = previewUrl;
            const virtualPlaylist = `#EXTM3U
#EXT-X-VERSION:3
#EXT-X-TARGETDURATION:10
#EXT-X-MEDIA-SEQUENCE:0
#EXTINF:10.0,
${tsUrl}
#EXT-X-ENDLIST`;
            
            console.log("Virtual playlist content:", virtualPlaylist);
            
            // Instead of blob URL, try using data URL or direct approach
            // Create a data URL for the playlist
            const dataUrl = `data:application/vnd.apple.mpegurl;charset=utf-8,${encodeURIComponent(virtualPlaylist)}`;
            console.log("Created data URL for playlist");
            
            // Destroy existing HLS instance if any
            if (hls) {
              hls.destroy();
            }
            
            // Create new HLS instance with TS-specific settings
            hls = new Hls({
              enableWorker: true,
              lowLatencyMode: true,
              backBufferLength: 30,
              maxBufferLength: 30,
              maxMaxBufferLength: 60,
              maxBufferSize: 60 * 1000 * 1000,
              startLevel: -1,
              // Important: allow loading of TS segments
              fragLoadingTimeOut: 20000,
              manifestLoadingTimeOut: 10000,
              xhrSetup: (xhr, url) => {
                xhr.withCredentials = false;
                // Set proper headers for TS stream
                xhr.setRequestHeader('Accept', 'video/mp2t, video/*, */*');
                console.log("Loading TS segment:", url);
              },
            });
            
            // Try loading with data URL first
            try {
              hls.loadSource(dataUrl);
              hls.attachMedia(video);
            } catch (err) {
              console.error("Failed to load with data URL, trying blob URL:", err);
              // Fallback to blob URL
              const blob = new Blob([virtualPlaylist], { type: 'application/vnd.apple.mpegurl' });
              blobUrl = URL.createObjectURL(blob);
              console.log("Created blob URL:", blobUrl);
              hls.loadSource(blobUrl);
              hls.attachMedia(video);
            }
            
            hls.on(Hls.Events.MANIFEST_PARSED, () => {
              console.log("HLS manifest parsed for TS stream, starting playback");
              console.log("Video readyState:", video.readyState);
              console.log("Video networkState:", video.networkState);
              video.play().catch((err) => {
                console.error("HLS video play error:", err);
                toast({
                  title: "Oynatma Hatası",
                  description: "TS stream oynatılamadı: " + err.message,
                  variant: "destructive",
                });
              });
            });
            
            hls.on(Hls.Events.LEVEL_LOADED, (event: any, data: any) => {
              console.log("HLS level loaded for TS stream:", data);
            });
            
            hls.on(Hls.Events.FRAG_LOADED, (event: any, data: any) => {
              console.log("TS fragment loaded:", data.frag.url, "Size:", data.frag.loaded);
            });
            
            hls.on(Hls.Events.FRAG_PARSING_INIT_SEGMENT, (event: any, data: any) => {
              console.log("TS fragment parsing init segment");
            });
            
            hls.on(Hls.Events.FRAG_PARSED, (event: any, data: any) => {
              console.log("TS fragment parsed successfully");
            });
            
            hls.on(Hls.Events.ERROR, (event: any, data: any) => {
              console.error("HLS error for TS stream:", data);
              console.error("Error details:", {
                type: data.type,
                details: data.details,
                fatal: data.fatal,
                url: data.url,
                response: data.response
              });
              
              if (data.fatal) {
                if (data.type === Hls.ErrorTypes.NETWORK_ERROR) {
                  console.log("Network error, retrying...");
                  setTimeout(() => {
                    if (hls) {
                      hls.startLoad();
                    }
                  }, 1000);
                } else if (data.type === Hls.ErrorTypes.MEDIA_ERROR) {
                  console.log("Media error, recovering...");
                  hls.recoverMediaError();
                } else {
                  console.error("Fatal HLS error:", data);
                  toast({
                    title: "Stream Hatası",
                    description: "TS stream yüklenirken hata: " + (data.details || data.message || "Bilinmeyen hata"),
                    variant: "destructive",
                  });
                }
              } else {
                console.warn("HLS non-fatal error:", data);
              }
            });
          } else {
            // Regular HLS playlist or try as HLS anyway
            console.log("Trying as HLS playlist or stream");
            hls.loadSource(previewUrl);
            hls.attachMedia(video);
          }
          
        } else if (video.canPlayType("application/vnd.apple.mpegurl")) {
          console.log("HLS.js not supported, using native HLS support (Safari)");
          // Native HLS support (Safari) - set src directly
          video.src = previewUrl;
          video.load();
          video.play().catch((err) => {
            console.error("Video play error:", err);
            toast({
              title: "Oynatma Hatası",
              description: "Video oynatılamadı: " + err.message,
              variant: "destructive",
            });
          });
        } else {
          console.log("HLS not supported, trying as regular video");
          // For non-HLS streams, try as regular video
          video.src = previewUrl;
          video.load();
          video.play().catch((err) => {
            console.error("Video play error:", err);
            toast({
              title: "Oynatma Hatası",
              description: "Video formatı desteklenmiyor veya URL erişilemiyor",
              variant: "destructive",
            });
          });
        }
      } catch (error: any) {
        console.error("Failed to initialize video player:", error);
        // Fallback to native HLS if available
        if (video.canPlayType("application/vnd.apple.mpegurl")) {
          console.log("Fallback: Using native HLS");
          video.src = previewUrl;
          video.load();
          video.play().catch((err) => {
            console.error("Video play error:", err);
          });
        } else {
          console.log("Fallback: Trying as regular video");
          // Try as regular video
          video.src = previewUrl;
          video.load();
          video.play().catch((err) => {
            console.error("Video play error:", err);
            toast({
              title: "Oynatma Hatası",
              description: "Video yüklenemedi: " + (error?.message || "Bilinmeyen hata"),
              variant: "destructive",
            });
          });
        }
      }
    };

      initHls();

      return () => {
        console.log("Cleaning up video player");
        if (hls) {
          hls.destroy();
          hls = null;
        }
        if (video) {
          video.pause();
          video.src = "";
          video.load();
        }
        // Clean up blob URL if it was created
        if (blobUrl) {
          URL.revokeObjectURL(blobUrl);
          blobUrl = null;
        }
      };
    };
    
    checkVideoRef();
  }, [getPreviewUrl, sourceUrl, channel?.status, channel?.output_url]);

  const fetchLogs = async () => {
    if (channel?.status !== "running") {
      toast({ title: "Uyarı", description: "Logları görmek için kanalın çalışıyor olması gerekir", variant: "destructive" });
      return;
    }

    setLoadingLogs(true);
    const result = await api.getChannelLogs(channelId);
    if (result.error) {
      toast({ title: "Hata", description: result.error, variant: "destructive" });
    } else if (result.data) {
      setLogs(result.data);
      setShowLogs(true);
    }
    setLoadingLogs(false);
  };

  const handleLogoUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;

    setUploading(true);
    const result = await api.uploadLogo(file);

    if (result.error) {
      toast({ title: "Hata", description: result.error, variant: "destructive" });
    } else if (result.data) {
      setLogoUrl(result.data.url); // Use relative path
      setLogoFilename(result.data.filename);
      setLogoPath(result.data.path);
      toast({ title: "Başarılı", description: "Logo yüklendi" });
    }
    setUploading(false);
  };

  const handleRemoveLogo = async () => {
    if (logoFilename) {
      await api.deleteLogo(logoFilename);
    }
    setLogoUrl(null);
    setLogoFilename(null);
    setLogoPath(null);
    setLogoX(10);
    setLogoY(10);
    toast({ title: "Başarılı", description: "Logo kaldırıldı" });
  };

  const handleSave = async () => {
    const wasRunning = channel?.status === "running";
    
    setSaving(true);

    // If channel is running, stop it first
    if (wasRunning) {
      toast({
        title: "Bilgi",
        description: "Kanal durduruluyor, güncelleme yapılıyor...",
      });
      
      const stopResult = await api.stopChannel(channelId);
      if (stopResult.error) {
        toast({ 
          title: "Hata", 
          description: "Kanal durdurulamadı: " + stopResult.error, 
          variant: "destructive" 
        });
        setSaving(false);
        return;
      }
      
      // Wait a bit for channel to stop
      await new Promise(resolve => setTimeout(resolve, 1000));
    }

    const logoConfig: LogoConfig | undefined = logoPath
      ? {
          path: logoPath,
          x: logoX,
          y: logoY,
          width: logoWidth,
          height: logoHeight,
          opacity: logoOpacity,
        }
      : undefined;

    const result = await api.updateChannel(channelId, {
      name,
      source_url: sourceUrl,
      logo: logoConfig,
      output_config: {
        codec: "libx264",
        bitrate,
        resolution: `${videoWidth}x${videoHeight}`,
        preset,
        profile: "high",
      },
    });

    if (result.error) {
      toast({ title: "Hata", description: result.error, variant: "destructive" });
      setSaving(false);
      return;
    }

    // If channel was running, restart it after a delay
    if (wasRunning) {
      toast({
        title: "Bilgi",
        description: "Kanal güncellendi, yeniden başlatılıyor...",
      });
      
      // Wait a bit before restarting
      await new Promise(resolve => setTimeout(resolve, 2000));
      
      const startResult = await api.startChannel(channelId);
      if (startResult.error) {
        toast({ 
          title: "Uyarı", 
          description: "Kanal güncellendi ancak otomatik başlatılamadı: " + startResult.error,
          variant: "destructive"
        });
      } else {
        toast({ 
          title: "Başarılı", 
          description: "Kanal güncellendi ve yeniden başlatıldı" 
        });
      }
    } else {
      toast({ title: "Başarılı", description: "Kanal güncellendi" });
    }

    // Refresh channel data
    const refreshResult = await api.getChannel(channelId);
    if (refreshResult.data) {
      setChannel(refreshResult.data);
    }

    setSaving(false);
  };

  // Drag handlers for logo positioning
  const handleMouseDown = useCallback(
    (e: React.MouseEvent) => {
      if (!previewRef.current || !logoUrl) return;
      setIsDragging(true);
      const rect = previewRef.current.getBoundingClientRect();
      setDragStart({
        x: e.clientX - (logoX / videoWidth) * rect.width,
        y: e.clientY - (logoY / videoHeight) * rect.height,
      });
    },
    [logoX, logoY, logoUrl, channel?.status]
  );

  const handleMouseMove = useCallback(
    (e: React.MouseEvent) => {
      if (!isDragging || !previewRef.current) return;
      const rect = previewRef.current.getBoundingClientRect();
      const scaleX = videoWidth / rect.width;
      const scaleY = videoHeight / rect.height;

      let newX = Math.round((e.clientX - dragStart.x) * scaleX);
      let newY = Math.round((e.clientY - dragStart.y) * scaleY);

      // Keep logo within bounds
      newX = Math.max(0, Math.min(newX, videoWidth - logoWidth));
      newY = Math.max(0, Math.min(newY, videoHeight - logoHeight));

      setLogoX(newX);
      setLogoY(newY);
    },
    [isDragging, dragStart, logoWidth, logoHeight]
  );

  const handleMouseUp = useCallback(() => {
    setIsDragging(false);
  }, []);

  const copyOutputUrl = () => {
    if (channel?.output_url) {
      navigator.clipboard.writeText(channel.output_url);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
      toast({ title: "Kopyalandı", description: "Yayın linki panoya kopyalandı" });
    }
  };

  const getOutputUrl = () => {
    const path = channel?.output_url || `/streams/${channelId}/index.m3u8`;
    return getFullUrl(path);
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <Loader2 className="w-8 h-8 animate-spin text-primary" />
      </div>
    );
  }

  if (!channel) {
    return null;
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div className="flex items-center gap-4">
          <Button
            variant="ghost"
            size="icon"
            onClick={() => router.push("/channels")}
          >
            <ArrowLeft className="w-5 h-5" />
          </Button>
          <div>
            <h1 className="text-3xl font-bold">Kanal Düzenle</h1>
            <p className="text-muted-foreground">{channel.name}</p>
          </div>
        </div>
        <Button onClick={handleSave} disabled={saving}>
          {saving ? (
            <>
              <Loader2 className="w-4 h-4 mr-2 animate-spin" />
              {channel?.status === "running" ? "Güncelleniyor..." : "Kaydediliyor..."}
            </>
          ) : (
            <>
              <Save className="w-4 h-4 mr-2" />
              Kaydet
            </>
          )}
        </Button>
        {channel?.status === "running" && (
          <p className="text-xs text-blue-500 mt-1">
            ℹ️ Aktif yayın var, kaydetme sırasında kanal durdurulup yeniden başlatılacak
          </p>
        )}
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Left Column - Video Preview & Logo Positioning */}
        <div className="space-y-6">
          {/* Video Preview with Logo Overlay */}
          <Card className="glass overflow-hidden">
            <CardHeader className="pb-3">
              <div className="flex items-center justify-between">
                <div>
                  <CardTitle className="flex items-center gap-2">
                    <Eye className="w-5 h-5" />
                    Önizleme
                  </CardTitle>
                  <CardDescription>
                    {channel?.status === "running"
                      ? "Canlı yayın önizlemesi - Logoyu sürükleyerek konumlandırın"
                      : "Yayını başlattığınızda önizleme burada görünecek"}
                  </CardDescription>
                </div>
                {sourceUrl && (
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => {
                      window.open(sourceUrl, "_blank");
                    }}
                  >
                    <ExternalLink className="w-4 h-4 mr-2" />
                    Kaynak URL'i Aç
                  </Button>
                )}
              </div>
            </CardHeader>
            <CardContent className="p-0">
              <div
                ref={previewRef}
                className="relative w-full bg-gradient-to-br from-slate-900 via-slate-800 to-slate-900 cursor-crosshair select-none overflow-hidden"
                style={{
                  aspectRatio: videoWidth > 0 && videoHeight > 0 ? `${videoWidth} / ${videoHeight}` : '16 / 9',
                }}
                onMouseMove={handleMouseMove}
                onMouseUp={handleMouseUp}
                onMouseLeave={handleMouseUp}
              >
                {/* Grid overlay */}
                <div className="absolute inset-0 opacity-10">
                  <div
                    className="w-full h-full"
                    style={{
                      backgroundImage:
                        "linear-gradient(rgba(255,255,255,0.1) 1px, transparent 1px), linear-gradient(90deg, rgba(255,255,255,0.1) 1px, transparent 1px)",
                      backgroundSize: "10% 10%",
                    }}
                  />
                </div>

                {/* Safe zone indicator */}
                <div className="absolute inset-[5%] border border-dashed border-white/20 rounded-lg" />

                {/* Center crosshair */}
                <div className="absolute left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2 w-8 h-8">
                  <div className="absolute left-1/2 top-0 bottom-0 w-px bg-white/20" />
                  <div className="absolute top-1/2 left-0 right-0 h-px bg-white/20" />
                </div>

                {/* Video Preview or Placeholder */}
                <video
                  ref={videoRef}
                  className={cn(
                    "absolute inset-0 w-full h-full object-contain",
                    getPreviewUrl() ? "block" : "hidden"
                  )}
                  autoPlay
                  muted
                  playsInline
                  controls={false}
                  crossOrigin="anonymous"
                  onError={(e) => {
                    console.error("Video load error:", e);
                    const target = e.target as HTMLVideoElement;
                    const error = target.error;
                    let errorMessage = "Video yüklenirken bir hata oluştu";
                    
                    if (error) {
                      switch (error.code) {
                        case error.MEDIA_ERR_ABORTED:
                          errorMessage = "Video yükleme iptal edildi";
                          break;
                        case error.MEDIA_ERR_NETWORK:
                          errorMessage = "Ağ hatası: Video yüklenemedi";
                          break;
                        case error.MEDIA_ERR_DECODE:
                          errorMessage = "Video çözümlenemedi (codec hatası)";
                          break;
                        case error.MEDIA_ERR_SRC_NOT_SUPPORTED:
                          errorMessage = "Video formatı desteklenmiyor veya URL erişilemiyor";
                          break;
                      }
                    }
                    
                    console.error("Video error details:", {
                      error: error,
                      errorCode: error?.code,
                      networkState: target.networkState,
                      readyState: target.readyState,
                      src: target.src,
                      currentSrc: target.currentSrc,
                    });
                    
                    toast({
                      title: "Video Yükleme Hatası",
                      description: errorMessage,
                      variant: "destructive",
                    });
                  }}
                  onLoadStart={() => {
                    console.log("Video load started");
                  }}
                  onLoadedMetadata={async (e) => {
                    const target = e.target as HTMLVideoElement;
                    const width = target.videoWidth;
                    const height = target.videoHeight;
                    console.log("Video metadata loaded", { width, height });
                    if (width > 0 && height > 0) {
                      setVideoWidth(width);
                      setVideoHeight(height);
                      
                      // Auto-save resolution to channel settings if channel is not running
                      if (channel && channel.status !== "running" && channelId) {
                        const currentResolution = channel.output_config?.resolution || "1920x1080";
                        const newResolution = `${width}x${height}`;
                        
                        // Only save if resolution changed
                        if (currentResolution !== newResolution) {
                          console.log("Auto-saving resolution:", newResolution);
                          try {
                            await api.updateChannel(channelId, {
                              name: channel.name,
                              source_url: channel.source_url,
                              logo: channel.logo,
                              output_config: {
                                codec: channel.output_config?.codec || "libx264",
                                bitrate: channel.output_config?.bitrate || "4000k",
                                resolution: newResolution,
                                preset: channel.output_config?.preset || "veryfast",
                                profile: channel.output_config?.profile || "high",
                              },
                            });
                            toast({
                              title: "Çözünürlük Kaydedildi",
                              description: `Kaynak çözünürlüğü (${newResolution}) otomatik olarak kaydedildi.`,
                              duration: 3000,
                            });
                            // Refresh channel data
                            const result = await api.getChannel(channelId);
                            if (result.data) {
                              setChannel(result.data);
                            }
                          } catch (error) {
                            console.error("Failed to auto-save resolution:", error);
                          }
                        }
                      }
                    }
                  }}
                  onCanPlay={() => {
                    console.log("Video can play");
                  }}
                  onWaiting={() => {
                    console.log("Video waiting for data");
                  }}
                  onStalled={() => {
                    console.log("Video stalled");
                  }}
                />
                {!getPreviewUrl() && (
                  <div className="absolute inset-0 flex items-center justify-center">
                    <div className="text-center">
                      <Play className="w-16 h-16 text-white/20 mx-auto mb-2" />
                      <p className="text-white/30 text-sm">
                        {sourceUrl
                          ? "Kaynak URL yükleniyor veya desteklenmeyen format..."
                          : "Kaynak URL girin (m3u8, ts veya diğer video formatları)"}
                      </p>
                      <p className="text-white/20 text-xs mt-1">1920 x 1080</p>
                      {sourceUrl && (
                        <p className="text-white/10 text-xs mt-2">
                          Kaynak: {sourceUrl.substring(0, 50)}...
                        </p>
                      )}
                    </div>
                  </div>
                )}

                {/* Draggable Logo */}
                {logoUrl && (
                  <motion.div
                    className={cn(
                      "absolute select-none cursor-move",
                      isDragging && "ring-2 ring-primary ring-offset-2 ring-offset-black"
                    )}
                    style={{
                      left: `${(logoX / videoWidth) * 100}%`,
                      top: `${(logoY / videoHeight) * 100}%`,
                      width: `${(logoWidth / videoWidth) * 100}%`,
                      height: `${(logoHeight / videoHeight) * 100}%`,
                      opacity: logoOpacity,
                    }}
                    onMouseDown={handleMouseDown}
                    whileHover={channel?.status !== "running" ? { scale: 1.02 } : {}}
                    transition={{ duration: 0.1 }}
                  >
                    <img
                      src={logoUrl}
                      alt="Logo"
                      className="w-full h-full pointer-events-none"
                      draggable={false}
                    />
                    <div className="absolute -bottom-6 left-0 right-0 text-center">
                      <span className="text-xs bg-black/70 text-white px-2 py-0.5 rounded">
                        <Move className="w-3 h-3 inline mr-1" />
                        Sürükle
                      </span>
                    </div>
                  </motion.div>
                )}

                {/* Position indicator */}
                {logoUrl && (
                  <div className="absolute bottom-2 right-2 bg-black/70 text-white text-xs px-2 py-1 rounded font-mono">
                    X: {logoX} | Y: {logoY}
                  </div>
                )}
              </div>
            </CardContent>
          </Card>

          {/* Logo Upload */}
          <Card className="glass">
            <CardHeader className="pb-3">
              <CardTitle className="flex items-center gap-2">
                <ImageIcon className="w-5 h-5" />
                Logo
              </CardTitle>
              <CardDescription>
                Yayın üzerine eklenecek logoyu yükleyin (PNG, JPG, GIF, WebP - max 5MB)
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              {logoUrl ? (
                <div className="space-y-4">
                  <div className="flex items-center gap-4 p-4 bg-secondary/50 rounded-lg">
                    <div className="w-20 h-20 bg-black/20 rounded-lg flex items-center justify-center overflow-hidden">
                      <img
                        src={logoUrl}
                        alt="Uploaded logo"
                        className="max-w-full max-h-full object-contain"
                      />
                    </div>
                    <div className="flex-1">
                      <p className="font-medium">Logo Yüklendi</p>
                      <p className="text-sm text-muted-foreground">
                        Önizleme alanında logoyu sürükleyerek konumlandırın
                      </p>
                    </div>
                    <Button
                      variant="destructive"
                      size="icon"
                      onClick={handleRemoveLogo}
                    >
                      <Trash2 className="w-4 h-4" />
                    </Button>
                  </div>

                  {/* Logo fine-tuning controls */}
                  <div className="grid grid-cols-2 gap-4">
                    <div className="space-y-2">
                      <Label htmlFor="logo-x">X Pozisyonu</Label>
                      <Input
                        id="logo-x"
                        type="number"
                        value={logoX}
                        onChange={(e) => setLogoX(Math.max(0, Number(e.target.value)))}
                        min={0}
                        max={videoWidth - logoWidth}
                      />
                    </div>
                    <div className="space-y-2">
                      <Label htmlFor="logo-y">Y Pozisyonu</Label>
                      <Input
                        id="logo-y"
                        type="number"
                        value={logoY}
                        onChange={(e) => setLogoY(Math.max(0, Number(e.target.value)))}
                        min={0}
                        max={videoHeight - logoHeight}
                      />
                    </div>
                  </div>
                  <div className="grid grid-cols-2 gap-4">
                    <div className="space-y-2">
                      <Label htmlFor="logo-width">Genişlik</Label>
                      <Input
                        id="logo-width"
                        type="number"
                        value={logoWidth}
                        onChange={(e) => setLogoWidth(Math.max(10, Number(e.target.value)))}
                        min={10}
                      />
                    </div>
                    <div className="space-y-2">
                      <Label htmlFor="logo-height">Yükseklik</Label>
                      <Input
                        id="logo-height"
                        type="number"
                        value={logoHeight}
                        onChange={(e) => setLogoHeight(Math.max(10, Number(e.target.value)))}
                        min={10}
                      />
                    </div>
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="logo-opacity">Opaklık: {logoOpacity.toFixed(1)}</Label>
                    <input
                      id="logo-opacity"
                      type="range"
                      min="0"
                      max="1"
                      step="0.1"
                      value={logoOpacity}
                      onChange={(e) => setLogoOpacity(Number(e.target.value))}
                      className="w-full h-2 bg-secondary rounded-lg appearance-none cursor-pointer accent-primary"
                    />
                  </div>
                </div>
              ) : (
                <label className="flex flex-col items-center justify-center w-full h-40 border-2 border-dashed border-border rounded-lg transition-colors cursor-pointer hover:border-primary/50 hover:bg-secondary/30">
                  <div className="flex flex-col items-center justify-center pt-5 pb-6">
                    {uploading ? (
                      <Loader2 className="w-10 h-10 mb-3 text-primary animate-spin" />
                    ) : (
                      <Upload className="w-10 h-10 mb-3 text-muted-foreground" />
                    )}
                    <p className="mb-2 text-sm text-muted-foreground">
                      <span className="font-semibold">Tıklayın</span> veya sürükleyin
                    </p>
                    <p className="text-xs text-muted-foreground">
                      PNG, JPG, GIF, WebP (max 5MB)
                    </p>
                  </div>
                  <input
                    type="file"
                    className="hidden"
                    accept=".png,.jpg,.jpeg,.gif,.webp"
                    onChange={handleLogoUpload}
                    disabled={uploading}
                  />
                </label>
              )}
            </CardContent>
          </Card>

        </div>

        {/* Right Column - Channel Settings */}
        <div className="space-y-6">
          {/* Stream URL */}
          <Card className="glass border-primary/30">
            <CardHeader className="pb-3">
              <CardTitle className="flex items-center gap-2">
                <LinkIcon className="w-5 h-5 text-primary" />
                Yayın Linki
              </CardTitle>
              <CardDescription>
                Bu linki kullanarak yayını izleyebilirsiniz
              </CardDescription>
            </CardHeader>
            <CardContent>
              <div className="flex gap-2">
                <Input
                  value={getOutputUrl()}
                  readOnly
                  className="font-mono text-sm bg-secondary/50"
                />
                <Button
                  variant="outline"
                  size="icon"
                  onClick={copyOutputUrl}
                  className="shrink-0"
                >
                  {copied ? (
                    <Check className="w-4 h-4 text-green-500" />
                  ) : (
                    <Copy className="w-4 h-4" />
                  )}
                </Button>
                <Button
                  variant="outline"
                  size="icon"
                  className="shrink-0"
                  onClick={() => window.open(getOutputUrl(), "_blank")}
                >
                  <ExternalLink className="w-4 h-4" />
                </Button>
              </div>
              {channel.status !== "running" && (
                <p className="text-xs text-amber-500 mt-2">
                  ⚠️ Yayını izlemek için kanalın çalışıyor olması gerekir
                </p>
              )}
            </CardContent>
          </Card>

          {/* Basic Settings */}
          <Card className="glass">
            <CardHeader className="pb-3">
              <CardTitle>Temel Ayarlar</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="name">Kanal Adı</Label>
                <Input
                  id="name"
                  placeholder="Kanalım"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="source">Kaynak URL</Label>
                <Input
                  id="source"
                  placeholder="rtmp://example.com/live/stream"
                  value={sourceUrl}
                  onChange={(e) => setSourceUrl(e.target.value)}
                />
              </div>
            </CardContent>
          </Card>

          {/* Encoding Settings */}
          <Card className="glass">
            <CardHeader className="pb-3">
              <CardTitle>Encoding Ayarları</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
                  <div className="grid grid-cols-2 gap-4">
                    <div className="space-y-2">
                      <Label htmlFor="bitrate">Bitrate</Label>
                      <Input
                        id="bitrate"
                        placeholder="4000k"
                        value={bitrate}
                        onChange={(e) => setBitrate(e.target.value)}
                      />
                    </div>
                    <div className="space-y-2">
                      <Label htmlFor="preset">Preset</Label>
                      <select
                        id="preset"
                        value={preset}
                        onChange={(e) => setPreset(e.target.value)}
                        className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background file:border-0 file:bg-transparent file:text-sm file:font-medium placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
                      >
                    <option value="ultrafast">Ultra Fast</option>
                    <option value="superfast">Super Fast</option>
                    <option value="veryfast">Very Fast</option>
                    <option value="faster">Faster</option>
                    <option value="fast">Fast</option>
                    <option value="medium">Medium</option>
                    <option value="slow">Slow</option>
                    <option value="slower">Slower</option>
                    <option value="veryslow">Very Slow</option>
                  </select>
                </div>
              </div>
              <div className="p-3 bg-secondary/50 rounded-lg">
                <p className="text-xs text-muted-foreground">
                  <strong>Codec:</strong> H.264 (libx264) | <strong>Çözünürlük:</strong> 1920x1080 | <strong>Profil:</strong> High
                </p>
              </div>
            </CardContent>
          </Card>

          {/* Actions */}
          <div className="flex gap-3">
            <Button
              variant="outline"
              className="flex-1"
              onClick={() => router.push("/channels")}
            >
              İptal
            </Button>
            <Button className="flex-1" onClick={handleSave} disabled={saving}>
              {saving ? (
                <Loader2 className="w-4 h-4 mr-2 animate-spin" />
              ) : (
                <Save className="w-4 h-4 mr-2" />
              )}
              Kaydet
            </Button>
          </div>
        </div>
      </div>

      {/* Log Viewer */}
      {channel && (
        <Card className="glass mt-6">
          <CardHeader className="pb-3">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                <Terminal className="w-5 h-5" />
                <CardTitle>FFmpeg Logları</CardTitle>
              </div>
              <div className="flex gap-2">
                {channel.status === "running" && (
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={fetchLogs}
                    disabled={loadingLogs}
                  >
                    {loadingLogs ? (
                      <Loader2 className="w-4 h-4 animate-spin" />
                    ) : (
                      <RefreshCw className="w-4 h-4" />
                    )}
                  </Button>
                )}
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setShowLogs(!showLogs)}
                >
                  {showLogs ? "Gizle" : "Göster"}
                </Button>
              </div>
            </div>
            <CardDescription>
              {channel.status === "running"
                ? "Canlı FFmpeg logları (otomatik güncelleniyor)"
                : "Kanal çalışırken loglar burada görünecek"}
            </CardDescription>
          </CardHeader>
          {showLogs && (
            <CardContent>
              {channel.status !== "running" ? (
                <div className="p-8 text-center text-muted-foreground">
                  <Terminal className="w-12 h-12 mx-auto mb-4 opacity-50" />
                  <p>Logları görmek için kanalı başlatın</p>
                </div>
              ) : logs.length === 0 ? (
                <div className="p-8 text-center text-muted-foreground">
                  <Loader2 className="w-8 h-8 mx-auto mb-4 animate-spin" />
                  <p>Loglar yükleniyor...</p>
                </div>
              ) : (
                <div className="relative">
                  <div className="h-96 overflow-y-auto bg-black/50 rounded-lg p-4 font-mono text-xs">
                    {logs.map((log, index) => {
                      const isError = log.toLowerCase().includes("error") || 
                                     log.toLowerCase().includes("failed") ||
                                     log.toLowerCase().includes("cannot");
                      const isWarning = log.toLowerCase().includes("warning");
                      
                      return (
                        <div
                          key={index}
                          className={cn(
                            "mb-1 break-words",
                            isError && "text-red-400",
                            isWarning && "text-yellow-400",
                            !isError && !isWarning && "text-green-400"
                          )}
                        >
                          {log}
                        </div>
                      );
                    })}
                    <div ref={logsEndRef} />
                  </div>
                  <div className="mt-2 flex items-center gap-4 text-xs text-muted-foreground">
                    <span>Toplam: {logs.length} satır</span>
                    <span className="flex items-center gap-1">
                      <span className="w-2 h-2 rounded-full bg-green-400" />
                      Bilgi
                    </span>
                    <span className="flex items-center gap-1">
                      <span className="w-2 h-2 rounded-full bg-yellow-400" />
                      Uyarı
                    </span>
                    <span className="flex items-center gap-1">
                      <span className="w-2 h-2 rounded-full bg-red-400" />
                      Hata
                    </span>
                  </div>
                </div>
              )}
            </CardContent>
          )}
        </Card>
      )}
    </div>
  );
}

