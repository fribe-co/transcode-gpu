"use client";

import { useState, useEffect } from "react";
import { motion } from "framer-motion";
import { Settings, Save, Loader2, Shield, Tv2, AlertCircle } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { useToast } from "@/hooks/use-toast";
import { api, Settings as SettingsType } from "@/lib/api";

export default function SettingsPage() {
  const { toast } = useToast();
  const [loading, setLoading] = useState(false);
  const [loadingSettings, setLoadingSettings] = useState(true);
  const [settings, setSettings] = useState<SettingsType | null>(null);
  const [hasRunningChannels, setHasRunningChannels] = useState(false);

  // Form state
  const [maxChannels, setMaxChannels] = useState(120);
  const [segmentTime, setSegmentTime] = useState(2);
  const [playlistSize, setPlaylistSize] = useState(6);
  const [logRetention, setLogRetention] = useState(7);
  const [defaultPreset, setDefaultPreset] = useState("veryfast");
  const [defaultBitrate, setDefaultBitrate] = useState("4000k");
  const [defaultResolution, setDefaultResolution] = useState("1920x1080");
  const [defaultProfile, setDefaultProfile] = useState("high");

  useEffect(() => {
    const fetchSettings = async () => {
      setLoadingSettings(true);
      const result = await api.getSettings();
      if (result.data) {
        setSettings(result.data);
        setMaxChannels(result.data.max_channels);
        setSegmentTime(result.data.segment_time);
        setPlaylistSize(result.data.playlist_size);
        setLogRetention(result.data.log_retention);
        setDefaultPreset(result.data.default_preset);
        setDefaultBitrate(result.data.default_bitrate);
        setDefaultResolution(result.data.default_resolution);
        setDefaultProfile(result.data.default_profile);
      }
      setLoadingSettings(false);
    };

    const checkRunningChannels = async () => {
      const result = await api.getChannels();
      if (result.data) {
        const running = result.data.some(c => c.status === "running");
        setHasRunningChannels(running);
      }
    };

    fetchSettings();
    checkRunningChannels();
  }, []);

  const handleSave = async () => {
    if (hasRunningChannels) {
      toast({
        title: "Hata",
        description: "Aktif yayın var, ayarlar güncellenemez. Lütfen önce tüm kanalları durdurun.",
        variant: "destructive",
      });
      return;
    }

    setLoading(true);
    try {
      const result = await api.updateSettings({
        max_channels: maxChannels,
        segment_time: segmentTime,
        playlist_size: playlistSize,
        log_retention: logRetention,
        default_preset: defaultPreset,
        default_bitrate: defaultBitrate,
        default_resolution: defaultResolution,
        default_profile: defaultProfile,
      });

      if (result.error) {
        toast({ title: "Hata", description: result.error, variant: "destructive" });
        setLoading(false);
        return;
      }

      // Reload settings from server to verify they were saved
      const refreshResult = await api.getSettings();
      if (refreshResult.data) {
        setSettings(refreshResult.data);
        setMaxChannels(refreshResult.data.max_channels);
        setSegmentTime(refreshResult.data.segment_time);
        setPlaylistSize(refreshResult.data.playlist_size);
        setLogRetention(refreshResult.data.log_retention);
        setDefaultPreset(refreshResult.data.default_preset);
        setDefaultBitrate(refreshResult.data.default_bitrate);
        setDefaultResolution(refreshResult.data.default_resolution);
        setDefaultProfile(refreshResult.data.default_profile);
        toast({ title: "Başarılı", description: "Ayarlar kaydedildi ve güncellendi" });
      } else {
        toast({ title: "Uyarı", description: "Ayarlar kaydedildi ancak yeniden yüklenemedi", variant: "destructive" });
      }
    } catch (error) {
      console.error("Settings update error:", error);
      toast({ title: "Hata", description: "Ayarlar kaydedilirken bir hata oluştu", variant: "destructive" });
    } finally {
      setLoading(false);
    }
  };

  if (loadingSettings) {
    return (
      <div className="flex items-center justify-center h-64">
        <Loader2 className="w-8 h-8 animate-spin text-primary" />
      </div>
    );
  }

  return (
    <div className="space-y-6 max-w-4xl">
      {/* Header */}
      <div>
        <h1 className="text-3xl font-bold">Ayarlar</h1>
        <p className="text-muted-foreground">CashbackTV yapılandırmanızı yönetin</p>
      </div>

      {/* Warning if channels are running */}
      {hasRunningChannels && (
        <motion.div
          initial={{ opacity: 0, y: -10 }}
          animate={{ opacity: 1, y: 0 }}
        >
          <Card className="glass border-amber-500/50 bg-amber-500/10">
            <CardContent className="pt-6">
              <div className="flex items-start gap-3">
                <AlertCircle className="w-5 h-5 text-amber-500 shrink-0 mt-0.5" />
                <div>
                  <p className="font-medium text-amber-500">Aktif Yayın Tespit Edildi</p>
                  <p className="text-sm text-muted-foreground mt-1">
                    Ayarları güncellemek için önce tüm kanalları durdurmanız gerekiyor.
                  </p>
                </div>
              </div>
            </CardContent>
          </Card>
        </motion.div>
      )}

      {/* General Settings */}
      <motion.div
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
      >
        <Card className="glass">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Settings className="w-5 h-5 text-primary" />
              Genel Ayarlar
            </CardTitle>
            <CardDescription>Genel uygulama ayarlarını yapılandırın</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label htmlFor="max-channels">Maksimum Kanal</Label>
                <Input
                  id="max-channels"
                  type="number"
                  value={maxChannels}
                  onChange={(e) => setMaxChannels(Number(e.target.value))}
                  disabled={hasRunningChannels}
                />
                <p className="text-xs text-muted-foreground">
                  Sistemde aynı anda çalışabilecek maksimum kanal sayısı. Bu limit aşıldığında yeni kanal başlatılamaz.
                </p>
              </div>
              <div className="space-y-2">
                <Label htmlFor="segment-time">HLS Segment Süresi (saniye)</Label>
                <Input
                  id="segment-time"
                  type="number"
                  value={segmentTime}
                  onChange={(e) => setSegmentTime(Number(e.target.value))}
                  disabled={hasRunningChannels}
                />
                <p className="text-xs text-muted-foreground">
                  Her HLS segment'inin süresi. Düşük değerler daha düşük gecikme sağlar ancak daha fazla segment üretir. Önerilen: 2-6 saniye.
                </p>
              </div>
              <div className="space-y-2">
                <Label htmlFor="playlist-size">Playlist Boyutu</Label>
                <Input
                  id="playlist-size"
                  type="number"
                  value={playlistSize}
                  onChange={(e) => setPlaylistSize(Number(e.target.value))}
                  disabled={hasRunningChannels}
                />
                <p className="text-xs text-muted-foreground">
                  HLS playlist'inde tutulacak segment sayısı. Daha fazla segment daha uzun geri sarma süresi sağlar ancak daha fazla depolama kullanır.
                </p>
              </div>
              <div className="space-y-2">
                <Label htmlFor="log-retention">Log Saklama Süresi (gün)</Label>
                <Input
                  id="log-retention"
                  type="number"
                  value={logRetention}
                  onChange={(e) => setLogRetention(Number(e.target.value))}
                  disabled={hasRunningChannels}
                />
                <p className="text-xs text-muted-foreground">
                  Sistem loglarının saklanacağı süre. Bu süre sonunda eski loglar otomatik olarak silinir.
                </p>
              </div>
            </div>
          </CardContent>
        </Card>
      </motion.div>

      {/* Encoding Defaults */}
      <motion.div
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ delay: 0.1 }}
      >
        <Card className="glass">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Tv2 className="w-5 h-5 text-primary" />
              Varsayılan Encoding Ayarları
            </CardTitle>
            <CardDescription>Yeni kanallar için varsayılan encoding parametrelerini ayarlayın</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label htmlFor="default-preset">Varsayılan Preset</Label>
                <select
                  id="default-preset"
                  value={defaultPreset}
                  onChange={(e) => setDefaultPreset(e.target.value)}
                  disabled={hasRunningChannels}
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
                <p className="text-xs text-muted-foreground">
                  FFmpeg encoding preset'i. Seçenekler: ultrafast, superfast, veryfast, faster, fast, medium, slow, slower, veryslow. Hızlı preset'ler daha az CPU kullanır ancak daha büyük dosya boyutları üretir.
                </p>
              </div>
              <div className="space-y-2">
                <Label htmlFor="default-bitrate">Varsayılan Bitrate</Label>
                <Input
                  id="default-bitrate"
                  value={defaultBitrate}
                  onChange={(e) => setDefaultBitrate(e.target.value)}
                  disabled={hasRunningChannels}
                />
                <p className="text-xs text-muted-foreground">
                  Çıktı video bitrate'i. Örnek: 4000k (4 Mbps), 8000k (8 Mbps). Daha yüksek bitrate daha iyi kalite sağlar ancak daha fazla bant genişliği kullanır.
                </p>
              </div>
              <div className="space-y-2">
                <Label htmlFor="default-resolution">Varsayılan Çözünürlük</Label>
                <Input
                  id="default-resolution"
                  value={defaultResolution}
                  onChange={(e) => setDefaultResolution(e.target.value)}
                  disabled={hasRunningChannels}
                />
                <p className="text-xs text-muted-foreground">
                  Çıktı video çözünürlüğü. Format: genişlikx yükseklik (örn: 1920x1080, 1280x720). Kaynak çözünürlükten daha yüksek olamaz.
                </p>
              </div>
              <div className="space-y-2">
                <Label htmlFor="default-profile">Varsayılan Profil</Label>
                <select
                  id="default-profile"
                  value={defaultProfile}
                  onChange={(e) => setDefaultProfile(e.target.value)}
                  disabled={hasRunningChannels}
                  className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background file:border-0 file:bg-transparent file:text-sm file:font-medium placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
                >
                  <option value="baseline">Baseline</option>
                  <option value="main">Main</option>
                  <option value="high">High</option>
                </select>
                <p className="text-xs text-muted-foreground">
                  H.264 codec profili. Seçenekler: baseline, main, high. High profili en iyi kaliteyi sağlar ancak daha fazla işlem gücü gerektirir.
                </p>
              </div>
            </div>
          </CardContent>
        </Card>
      </motion.div>


      {/* Save Button */}
      <div className="flex justify-end">
        <Button onClick={handleSave} disabled={loading || hasRunningChannels}>
          {loading ? (
            <>
              <Loader2 className="w-4 h-4 mr-2 animate-spin" />
              Kaydediliyor...
            </>
          ) : (
            <>
              <Save className="w-4 h-4 mr-2" />
              Ayarları Kaydet
            </>
          )}
        </Button>
      </div>
    </div>
  );
}

