"use client";

import { useEffect, useState } from "react";
import { motion } from "framer-motion";
import {
  Activity,
  Cpu,
  HardDrive,
  Wifi,
  MemoryStick,
  Clock,
  Radio,
  Loader2,
  Thermometer,
} from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Progress } from "@/components/ui/progress";
import { api, Channel, ProcessMetrics, SystemInfo } from "@/lib/api";
import { formatBytes, formatDuration, cn, getStatusDotClass, getStatusText } from "@/lib/utils";

export default function MonitoringPage() {
  const [channels, setChannels] = useState<Channel[]>([]);
  const [metrics, setMetrics] = useState<Record<string, ProcessMetrics>>({});
  const [systemInfo, setSystemInfo] = useState<SystemInfo | null>(null);
  const [loading, setLoading] = useState(true);

  const fetchData = async () => {
    try {
      // Fetch system info and channels in parallel
      const [systemResult, channelsResult, metricsResult] = await Promise.all([
        api.getSystemInfo(),
        api.getChannels(),
        api.getAllChannelMetrics(), // Optimized: single request for all metrics
      ]);

      if (systemResult.data) {
        setSystemInfo(systemResult.data);
      }

      if (channelsResult.data) {
        setChannels(channelsResult.data);
      }

      // Convert metrics array to map for easier lookup
      if (metricsResult.data) {
        const metricsMap: Record<string, ProcessMetrics> = {};
        metricsResult.data.forEach((metric) => {
          if (metric && metric.channel_id) {
            metricsMap[metric.channel_id] = metric;
          }
        });
        setMetrics(metricsMap);
      }
    } catch (error) {
      console.error("Error fetching monitoring data:", error);
      // Don't block UI on error, keep showing previous data
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchData();
    // Poll every 3 seconds for more real-time updates
    // System info is cached for 5 seconds on backend, so 3s provides good balance
    const interval = setInterval(fetchData, 3000);
    return () => clearInterval(interval);
  }, []);

  const runningChannels = channels.filter(c => c.status === "running");
  // Use system CPU usage if available, otherwise sum process CPU
  const totalCPU = systemInfo?.cpu_usage || Object.values(metrics).reduce((sum, m) => sum + (m?.cpu_usage || 0), 0);
  // Use system memory if available
  const totalMemory = systemInfo?.memory_used || Object.values(metrics).reduce((sum, m) => sum + (m?.memory_usage || 0), 0);
  const memoryTotal = systemInfo?.memory_total || 256 * 1024 * 1024 * 1024; // Default 256GB if not available

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <Loader2 className="w-8 h-8 animate-spin text-primary" />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-3xl font-bold">İzleme</h1>
        <p className="text-muted-foreground">Gerçek zamanlı sistem ve kanal metrikleri</p>
      </div>

      {/* System Information */}
      {systemInfo && (
        <Card className="glass border-primary/50">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <HardDrive className="w-5 h-5 text-primary" />
              Makine Bilgileri
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
              <div>
                <p className="text-xs text-muted-foreground mb-1">CPU Core</p>
                <p className="text-lg font-bold">{systemInfo.cpu_cores}</p>
              </div>
              <div>
                <p className="text-xs text-muted-foreground mb-1">CPU Thread</p>
                <p className="text-lg font-bold">{systemInfo.cpu_threads}</p>
              </div>
              <div>
                <p className="text-xs text-muted-foreground mb-1">Load Average (1m)</p>
                <p className="text-lg font-bold">{systemInfo.load_average_1.toFixed(2)}</p>
              </div>
              <div>
                <p className="text-xs text-muted-foreground mb-1">Uptime</p>
                <p className="text-lg font-bold">{formatDuration(systemInfo.uptime)}</p>
              </div>
            </div>
          </CardContent>
        </Card>
      )}

      {/* GPU Information */}
      {systemInfo && systemInfo.gpus && systemInfo.gpus.length > 0 && (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {systemInfo.gpus.map((gpu) => (
            <Card key={gpu.id} className="glass border-success/30">
              <CardHeader className="pb-2">
                <CardTitle className="text-sm font-medium flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <div className="p-1 rounded bg-success/10">
                      <Cpu className="w-4 h-4 text-success" />
                    </div>
                    <span>GPU {gpu.id}: {gpu.name}</span>
                  </div>
                  <div className="flex items-center gap-1 text-xs font-normal text-muted-foreground">
                    <Thermometer className={cn(
                      "w-3 h-3",
                      gpu.temperature > 70 ? "text-destructive" : "text-success"
                    )} />
                    {gpu.temperature}°C
                  </div>
                </CardTitle>
              </CardHeader>
              <CardContent>
                <div className="space-y-3">
                  <div>
                    <div className="flex items-center justify-between text-xs mb-1">
                      <span className="text-muted-foreground">GPU Kullanımı</span>
                      <span className="font-medium">{gpu.utilization.toFixed(1)}%</span>
                    </div>
                    <Progress value={gpu.utilization} className="h-1.5" />
                  </div>
                  <div>
                    <div className="flex items-center justify-between text-xs mb-1">
                      <span className="text-muted-foreground">VRAM</span>
                      <span className="font-medium">
                        {formatBytes(gpu.memory_used)} / {formatBytes(gpu.memory_total)}
                      </span>
                    </div>
                    <Progress value={(gpu.memory_used / gpu.memory_total) * 100} className="h-1.5" />
                  </div>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      )}

      {/* System Overview */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        <Card className="glass">
          <CardContent className="p-4">
            <div className="flex items-center gap-3">
              <div className="p-2 rounded-lg bg-primary/10">
                <Cpu className="w-5 h-5 text-primary" />
              </div>
              <div className="flex-1">
                <p className="text-2xl font-bold">{totalCPU.toFixed(2)}%</p>
                <p className="text-xs text-muted-foreground">
                  {systemInfo ? `Sistem CPU (${systemInfo.cpu_cores} core)` : "Toplam CPU Kullanımı"}
                </p>
              </div>
            </div>
            <Progress value={totalCPU} className="mt-3 h-1" />
          </CardContent>
        </Card>

        <Card className="glass">
          <CardContent className="p-4">
            <div className="flex items-center gap-3">
              <div className="p-2 rounded-lg bg-blue-500/10">
                <MemoryStick className="w-5 h-5 text-blue-500" />
              </div>
              <div className="flex-1">
                <p className="text-2xl font-bold">{formatBytes(totalMemory)}</p>
                <p className="text-xs text-muted-foreground">
                  {systemInfo ? `${systemInfo.memory_percent.toFixed(1)}% kullanılıyor` : "Toplam Bellek"}
                </p>
                {systemInfo && (
                  <p className="text-xs text-muted-foreground mt-1">
                    {formatBytes(systemInfo.memory_total)} toplam
                  </p>
                )}
              </div>
            </div>
            <Progress value={systemInfo ? systemInfo.memory_percent : (totalMemory / memoryTotal) * 100} className="mt-3 h-1" />
          </CardContent>
        </Card>

        <Card className="glass">
          <CardContent className="p-4">
            <div className="flex items-center gap-3">
              <div className="p-2 rounded-lg bg-success/10">
                <Radio className="w-5 h-5 text-success" />
              </div>
              <div>
                <p className="text-2xl font-bold">{runningChannels.length}</p>
                <p className="text-xs text-muted-foreground">Aktif Yayınlar</p>
              </div>
            </div>
          </CardContent>
        </Card>

        <Card className="glass">
          <CardContent className="p-4">
            <div className="flex items-center gap-3">
              <div className="p-2 rounded-lg bg-warning/10">
                <Activity className="w-5 h-5 text-warning" />
              </div>
              <div>
                <p className="text-2xl font-bold">
                  {Object.values(metrics).reduce((sum, m) => sum + (m?.fps || 0), 0).toFixed(0)}
                </p>
                <p className="text-xs text-muted-foreground">Toplam FPS</p>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Active Processes */}
      <Card className="glass">
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Activity className="w-5 h-5 text-primary" />
            Aktif Transcoding İşlemleri
          </CardTitle>
        </CardHeader>
        <CardContent>
          {runningChannels.length === 0 ? (
            <div className="text-center py-8 text-muted-foreground">
              <Radio className="w-12 h-12 mx-auto mb-4 opacity-50" />
              <p>Aktif transcoding işlemi yok</p>
              <p className="text-sm">Gerçek zamanlı metrikleri görmek için bir kanal başlatın</p>
            </div>
          ) : (
            <div className="space-y-4">
              {runningChannels.map((channel, index) => {
                const m = metrics[channel.id];
                return (
                  <motion.div
                    key={channel.id}
                    initial={{ opacity: 0, x: -20 }}
                    animate={{ opacity: 1, x: 0 }}
                    transition={{ delay: index * 0.1 }}
                    className="p-4 rounded-lg bg-secondary/50 border border-border"
                  >
                    <div className="flex items-center justify-between mb-4">
                      <div className="flex items-center gap-3">
                        <div className={cn("status-dot", getStatusDotClass(channel.status))} />
                        <div>
                          <h3 className="font-medium">{channel.name}</h3>
                          {m && (
                            <p className="text-xs text-muted-foreground">
                              PID: {m.pid} | Uptime: {formatDuration(m.uptime)}
                            </p>
                          )}
                        </div>
                      </div>
                      {m && (
                        <div className="text-right">
                          <p className="text-lg font-bold text-success">{m.speed.toFixed(2)}x</p>
                          <p className="text-xs text-muted-foreground">Hız</p>
                        </div>
                      )}
                    </div>
                    {m ? (
                      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                        <div>
                          <p className="text-xs text-muted-foreground mb-1">CPU</p>
                          <div className="flex items-center gap-2">
                            <Progress value={m.cpu_usage} className="h-2 flex-1" />
                            <span className="text-sm font-medium">{m.cpu_usage.toFixed(1)}%</span>
                          </div>
                        </div>
                        <div>
                          <p className="text-xs text-muted-foreground mb-1">Memory</p>
                          <p className="text-sm font-medium">{formatBytes(m.memory_usage)}</p>
                        </div>
                        <div>
                          <p className="text-xs text-muted-foreground mb-1">FPS</p>
                          <p className="text-sm font-medium">{m.fps.toFixed(1)}</p>
                        </div>
                        <div>
                          <p className="text-xs text-muted-foreground mb-1">Düşen</p>
                          <p className={cn(
                            "text-sm font-medium",
                            m.dropped_frames > 0 ? "text-destructive" : "text-success"
                          )}>
                            {m.dropped_frames}
                          </p>
                        </div>
                      </div>
                    ) : (
                      <div className="flex items-center justify-center py-4">
                        <Loader2 className="w-5 h-5 animate-spin text-muted-foreground" />
                        <span className="ml-2 text-sm text-muted-foreground">Metrikler yükleniyor...</span>
                      </div>
                    )}
                  </motion.div>
                );
              })}
            </div>
          )}
        </CardContent>
      </Card>

      {/* All Channels Status */}
      <Card className="glass">
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Radio className="w-5 h-5 text-primary" />
            Tüm Kanalların Durumu
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-2 md:grid-cols-4 lg:grid-cols-6 gap-2">
            {channels.map((channel) => (
              <div
                key={channel.id}
                className="p-3 rounded-lg bg-secondary/50 border border-border text-center"
              >
                <div className={cn("status-dot mx-auto mb-2", getStatusDotClass(channel.status))} />
                <p className="text-sm font-medium truncate">{channel.name}</p>
                <p className="text-xs text-muted-foreground capitalize">{getStatusText(channel.status)}</p>
              </div>
            ))}
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

