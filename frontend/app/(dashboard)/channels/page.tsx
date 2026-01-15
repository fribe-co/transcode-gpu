"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { motion } from "framer-motion";
import {
  Plus,
  Play,
  Square,
  RotateCcw,
  Trash2,
  Edit,
  Radio,
  Loader2,
  Search,
  LayoutGrid,
  List,
  ExternalLink,
  Link as LinkIcon,
  CheckSquare,
  Square as SquareIcon,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { api, Channel } from "@/lib/api";
import { useToast } from "@/hooks/use-toast";
import { cn, getStatusColor, getStatusDotClass, formatDuration, getStatusText } from "@/lib/utils";
import { ChannelDialog } from "@/components/channels/channel-dialog";

export default function ChannelsPage() {
  const router = useRouter();
  const { toast } = useToast();
  const [channels, setChannels] = useState<Channel[]>([]);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState("");
  const [view, setView] = useState<"grid" | "list">("grid");
  const [dialogOpen, setDialogOpen] = useState(false);
  const [actionLoading, setActionLoading] = useState<string | null>(null);
  const [selectedChannels, setSelectedChannels] = useState<Set<string>>(new Set());
  const [batchLoading, setBatchLoading] = useState(false);

  const getStreamUrl = (channel: Channel) => {
    // output_url comes from API, if it's already a full URL use it, otherwise use CDN
    if (channel.output_url) {
      if (channel.output_url.startsWith("http://") || channel.output_url.startsWith("https://")) {
        return channel.output_url;
      }
      return `https://cdn.cashbacktv.live${channel.output_url}`;
    }
    return `https://cdn.cashbacktv.live/streams/${channel.id}/index.m3u8`;
  };

  const fetchChannels = async () => {
    const result = await api.getChannels();
    if (result.data) {
      setChannels(result.data);
    }
    setLoading(false);
  };

  useEffect(() => {
    fetchChannels();
    const interval = setInterval(fetchChannels, 5000);
    return () => clearInterval(interval);
  }, []);

  const handleStart = async (id: string) => {
    setActionLoading(id);
    const result = await api.startChannel(id);
    if (result.error) {
      toast({ title: "Hata", description: result.error, variant: "destructive" });
    } else {
      toast({ title: "Başarılı", description: "Kanal başlatıldı" });
      fetchChannels();
    }
    setActionLoading(null);
  };

  const handleStop = async (id: string) => {
    setActionLoading(id);
    const result = await api.stopChannel(id);
    if (result.error) {
      toast({ title: "Hata", description: result.error, variant: "destructive" });
    } else {
      toast({ title: "Başarılı", description: "Kanal durduruldu" });
      fetchChannels();
    }
    setActionLoading(null);
  };

  const handleRestart = async (id: string) => {
    setActionLoading(id);
    const result = await api.restartChannel(id);
    if (result.error) {
      toast({ title: "Hata", description: result.error, variant: "destructive" });
    } else {
      toast({ title: "Başarılı", description: "Kanal yeniden başlatıldı" });
      fetchChannels();
    }
    setActionLoading(null);
  };

  const handleDelete = async (id: string) => {
    if (!confirm("Bu kanalı silmek istediğinizden emin misiniz?")) return;
    
    setActionLoading(id);
    const result = await api.deleteChannel(id);
    if (result.error) {
      toast({ title: "Hata", description: result.error, variant: "destructive" });
    } else {
      toast({ title: "Başarılı", description: "Kanal silindi" });
      fetchChannels();
    }
    setActionLoading(null);
  };

  // Batch operations
  const toggleChannelSelection = (id: string) => {
    const newSelected = new Set(selectedChannels);
    if (newSelected.has(id)) {
      newSelected.delete(id);
    } else {
      newSelected.add(id);
    }
    setSelectedChannels(newSelected);
  };

  const toggleSelectAll = () => {
    if (selectedChannels.size === filteredChannels.length) {
      setSelectedChannels(new Set());
    } else {
      setSelectedChannels(new Set(filteredChannels.map((c) => c.id)));
    }
  };

  const handleBatchStart = async () => {
    if (selectedChannels.size === 0) return;
    
    setBatchLoading(true);
    const result = await api.batchStartChannels(Array.from(selectedChannels));
    if (result.error) {
      toast({ title: "Hata", description: result.error, variant: "destructive" });
    } else if (result.data) {
      const successCount = result.data.success.length;
      const failedCount = result.data.failed.length;
      if (failedCount > 0) {
        toast({
          title: "Kısmi Başarı",
          description: `${successCount} kanal başlatıldı, ${failedCount} kanal başlatılamadı`,
          variant: "default",
        });
      } else {
        toast({ title: "Başarılı", description: `${successCount} kanal başlatıldı` });
      }
      setSelectedChannels(new Set());
      fetchChannels();
    }
    setBatchLoading(false);
  };

  const handleBatchStop = async () => {
    if (selectedChannels.size === 0) return;
    
    setBatchLoading(true);
    const result = await api.batchStopChannels(Array.from(selectedChannels));
    if (result.error) {
      toast({ title: "Hata", description: result.error, variant: "destructive" });
    } else if (result.data) {
      const successCount = result.data.success.length;
      const failedCount = result.data.failed.length;
      if (failedCount > 0) {
        toast({
          title: "Kısmi Başarı",
          description: `${successCount} kanal durduruldu, ${failedCount} kanal durdurulamadı`,
          variant: "default",
        });
      } else {
        toast({ title: "Başarılı", description: `${successCount} kanal durduruldu` });
      }
      setSelectedChannels(new Set());
      fetchChannels();
    }
    setBatchLoading(false);
  };

  const handleBatchRestart = async () => {
    if (selectedChannels.size === 0) return;
    
    setBatchLoading(true);
    const result = await api.batchRestartChannels(Array.from(selectedChannels));
    if (result.error) {
      toast({ title: "Hata", description: result.error, variant: "destructive" });
    } else if (result.data) {
      const successCount = result.data.success.length;
      const failedCount = result.data.failed.length;
      if (failedCount > 0) {
        toast({
          title: "Kısmi Başarı",
          description: `${successCount} kanal yeniden başlatıldı, ${failedCount} kanal yeniden başlatılamadı`,
          variant: "default",
        });
      } else {
        toast({ title: "Başarılı", description: `${successCount} kanal yeniden başlatıldı` });
      }
      setSelectedChannels(new Set());
      fetchChannels();
    }
    setBatchLoading(false);
  };

  const handleBatchDelete = async () => {
    if (selectedChannels.size === 0) return;
    if (!confirm(`${selectedChannels.size} kanalı silmek istediğinizden emin misiniz?`)) return;
    
    setBatchLoading(true);
    const result = await api.batchDeleteChannels(Array.from(selectedChannels));
    if (result.error) {
      toast({ title: "Hata", description: result.error, variant: "destructive" });
    } else if (result.data) {
      const successCount = result.data.success.length;
      const failedCount = result.data.failed.length;
      if (failedCount > 0) {
        toast({
          title: "Kısmi Başarı",
          description: `${successCount} kanal silindi, ${failedCount} kanal silinemedi`,
          variant: "default",
        });
      } else {
        toast({ title: "Başarılı", description: `${successCount} kanal silindi` });
      }
      setSelectedChannels(new Set());
      fetchChannels();
    }
    setBatchLoading(false);
  };

  const filteredChannels = channels.filter((c) =>
    c.name.toLowerCase().includes(search.toLowerCase())
  );

  const stats = {
    total: channels.length,
    running: channels.filter((c) => c.status === "running").length,
    stopped: channels.filter((c) => c.status === "stopped").length,
    error: channels.filter((c) => c.status === "error").length,
  };

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
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h1 className="text-3xl font-bold">Kanallar</h1>
          <p className="text-muted-foreground">Video transcoding kanallarınızı yönetin</p>
        </div>
        <Button onClick={() => setDialogOpen(true)}>
          <Plus className="w-4 h-4 mr-2" />
          Kanal Ekle
        </Button>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <Card className="glass">
          <CardContent className="p-4">
            <div className="flex items-center gap-3">
              <div className="p-2 rounded-lg bg-primary/10">
                <Radio className="w-5 h-5 text-primary" />
              </div>
              <div>
                <p className="text-2xl font-bold">{stats.total}</p>
                <p className="text-xs text-muted-foreground">Toplam Kanal</p>
              </div>
            </div>
          </CardContent>
        </Card>
        <Card className="glass">
          <CardContent className="p-4">
            <div className="flex items-center gap-3">
              <div className="p-2 rounded-lg bg-success/10">
                <Play className="w-5 h-5 text-success" />
              </div>
              <div>
                <p className="text-2xl font-bold">{stats.running}</p>
                <p className="text-xs text-muted-foreground">Çalışıyor</p>
              </div>
            </div>
          </CardContent>
        </Card>
        <Card className="glass">
          <CardContent className="p-4">
            <div className="flex items-center gap-3">
              <div className="p-2 rounded-lg bg-muted">
                <Square className="w-5 h-5 text-muted-foreground" />
              </div>
              <div>
                <p className="text-2xl font-bold">{stats.stopped}</p>
                <p className="text-xs text-muted-foreground">Durduruldu</p>
              </div>
            </div>
          </CardContent>
        </Card>
        <Card className="glass">
          <CardContent className="p-4">
            <div className="flex items-center gap-3">
              <div className="p-2 rounded-lg bg-destructive/10">
                <Radio className="w-5 h-5 text-destructive" />
              </div>
              <div>
                <p className="text-2xl font-bold">{stats.error}</p>
                <p className="text-xs text-muted-foreground">Hatalar</p>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Filters */}
      <div className="flex flex-col sm:flex-row gap-4">
        <div className="relative flex-1">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
          <Input
            placeholder="Kanallarda ara..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="pl-10"
          />
        </div>
        <div className="flex gap-1 p-1 bg-secondary rounded-lg">
          <Button
            variant={view === "grid" ? "default" : "ghost"}
            size="sm"
            onClick={() => setView("grid")}
          >
            <LayoutGrid className="w-4 h-4" />
          </Button>
          <Button
            variant={view === "list" ? "default" : "ghost"}
            size="sm"
            onClick={() => setView("list")}
          >
            <List className="w-4 h-4" />
          </Button>
        </div>
      </div>

      {/* Batch Actions Toolbar - Always Visible */}
      <Card className={cn("glass shadow-lg transition-colors", selectedChannels.size > 0 && "border-primary/50")}>
        <CardContent className="p-4">
          <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4">
            <div className="flex items-center gap-4">
              <Button
                variant="outline"
                size="sm"
                onClick={toggleSelectAll}
                className="flex items-center gap-2"
              >
                {selectedChannels.size === filteredChannels.length && filteredChannels.length > 0 ? (
                  <CheckSquare className="w-4 h-4" />
                ) : (
                  <SquareIcon className="w-4 h-4" />
                )}
                <span className="text-sm">
                  {selectedChannels.size === filteredChannels.length && filteredChannels.length > 0
                    ? "Tümünü Kaldır"
                    : "Tümünü Seç"}
                </span>
              </Button>
              {selectedChannels.size > 0 ? (
                <span className="text-sm font-medium text-primary">
                  {selectedChannels.size} kanal seçildi
                </span>
              ) : (
                <span className="text-sm text-muted-foreground">
                  Toplu işlem için kanal seçin
                </span>
              )}
            </div>
            <div className="flex flex-wrap gap-2 w-full sm:w-auto">
              <Button
                variant="success"
                size="sm"
                onClick={handleBatchStart}
                disabled={batchLoading || selectedChannels.size === 0}
                className="flex-1 sm:flex-initial"
              >
                {batchLoading ? (
                  <Loader2 className="w-4 h-4 animate-spin" />
                ) : (
                  <>
                    <Play className="w-4 h-4 mr-2" />
                    Başlat
                  </>
                )}
              </Button>
              <Button
                variant="outline"
                size="sm"
                onClick={handleBatchStop}
                disabled={batchLoading || selectedChannels.size === 0}
                className="flex-1 sm:flex-initial"
              >
                {batchLoading ? (
                  <Loader2 className="w-4 h-4 animate-spin" />
                ) : (
                  <>
                    <Square className="w-4 h-4 mr-2" />
                    Durdur
                  </>
                )}
              </Button>
              <Button
                variant="outline"
                size="sm"
                onClick={handleBatchRestart}
                disabled={batchLoading || selectedChannels.size === 0}
                className="flex-1 sm:flex-initial border-primary/50 hover:border-primary hover:bg-primary/10"
              >
                {batchLoading ? (
                  <Loader2 className="w-4 h-4 animate-spin" />
                ) : (
                  <>
                    <RotateCcw className="w-4 h-4 mr-2" />
                    Yeniden Başlat
                  </>
                )}
              </Button>
              <Button
                variant="destructive"
                size="sm"
                onClick={handleBatchDelete}
                disabled={batchLoading || selectedChannels.size === 0}
                className="flex-1 sm:flex-initial"
              >
                {batchLoading ? (
                  <Loader2 className="w-4 h-4 animate-spin" />
                ) : (
                  <>
                    <Trash2 className="w-4 h-4 mr-2" />
                    Sil
                  </>
                )}
              </Button>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Channels Grid/List */}
      {filteredChannels.length === 0 ? (
        <Card className="glass">
          <CardContent className="py-12 text-center">
            <Radio className="w-12 h-12 mx-auto mb-4 text-muted-foreground" />
            <h3 className="text-lg font-medium mb-2">Kanal bulunamadı</h3>
            <p className="text-muted-foreground mb-4">
              {search ? "Farklı bir arama terimi deneyin" : "Başlamak için ilk kanalınızı oluşturun"}
            </p>
            {!search && (
              <Button onClick={() => setDialogOpen(true)}>
                <Plus className="w-4 h-4 mr-2" />
                Kanal Ekle
              </Button>
            )}
          </CardContent>
        </Card>
      ) : view === "grid" ? (
        <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
          {filteredChannels.map((channel, index) => (
            <motion.div
              key={channel.id}
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ delay: index * 0.05 }}
            >
              <Card className={cn("glass hover:border-primary/50 transition-colors", selectedChannels.has(channel.id) && "border-primary")}>
                <CardHeader className="pb-3">
                  <div className="flex items-start justify-between">
                    <div className="flex items-center gap-3">
                      <button
                        onClick={() => toggleChannelSelection(channel.id)}
                        className="mt-1"
                        type="button"
                      >
                        {selectedChannels.has(channel.id) ? (
                          <CheckSquare className="w-4 h-4 text-primary" />
                        ) : (
                          <SquareIcon className="w-4 h-4 text-muted-foreground" />
                        )}
                      </button>
                      <div className={cn("status-dot", getStatusDotClass(channel.status))} />
                      <div>
                        <CardTitle className="text-base">{channel.name}</CardTitle>
                        <p className={cn("text-sm capitalize", getStatusColor(channel.status))}>
                          {getStatusText(channel.status)}
                        </p>
                      </div>
                    </div>
                    <div className="flex gap-1">
                      <Button
                        variant="ghost"
                        size="icon"
                        className="h-8 w-8"
                        onClick={() => router.push(`/channels/${channel.id}/edit`)}
                        title="Düzenle"
                      >
                        <Edit className="w-4 h-4" />
                      </Button>
                      <Button
                        variant="ghost"
                        size="icon"
                        className="h-8 w-8 text-destructive hover:text-destructive"
                        onClick={() => handleDelete(channel.id)}
                        disabled={actionLoading === channel.id}
                        title="Sil"
                      >
                        <Trash2 className="w-4 h-4" />
                      </Button>
                    </div>
                  </div>
                </CardHeader>
                <CardContent className="pt-0">
                  <p className="text-xs text-muted-foreground truncate mb-2">
                    {channel.source_url}
                  </p>
                  
                  {/* Stream Link */}
                  {channel.status === "running" && (
                    <div className="flex items-center gap-2 mb-4 p-2 bg-success/10 rounded-lg">
                      <LinkIcon className="w-3 h-3 text-success shrink-0" />
                      <span className="text-xs text-success truncate flex-1 font-mono">
                        {getStreamUrl(channel)}
                      </span>
                      <Button
                        variant="ghost"
                        size="icon"
                        className="h-6 w-6 shrink-0"
                        onClick={() => window.open(getStreamUrl(channel), "_blank")}
                        title="Yayını İzle"
                      >
                        <ExternalLink className="w-3 h-3" />
                      </Button>
                    </div>
                  )}
                  {channel.status !== "running" && (
                    <div className="flex items-center gap-2 mb-4 p-2 bg-secondary/50 rounded-lg">
                      <LinkIcon className="w-3 h-3 text-muted-foreground shrink-0" />
                      <span className="text-xs text-muted-foreground">
                        Yayın linki için kanalı başlatın
                      </span>
                    </div>
                  )}

                  <div className="flex gap-2">
                    {channel.status === "running" ? (
                      <>
                        <Button
                          variant="outline"
                          size="sm"
                          className="flex-1"
                          onClick={() => handleStop(channel.id)}
                          disabled={actionLoading === channel.id}
                        >
                          {actionLoading === channel.id ? (
                            <Loader2 className="w-4 h-4 animate-spin" />
                          ) : (
                            <>
                              <Square className="w-4 h-4 mr-2" />
                              Durdur
                            </>
                          )}
                        </Button>
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() => handleRestart(channel.id)}
                          disabled={actionLoading === channel.id}
                        >
                          <RotateCcw className="w-4 h-4" />
                        </Button>
                      </>
                    ) : (
                      <Button
                        variant="success"
                        size="sm"
                        className="flex-1"
                        onClick={() => handleStart(channel.id)}
                        disabled={actionLoading === channel.id}
                      >
                        {actionLoading === channel.id ? (
                          <Loader2 className="w-4 h-4 animate-spin" />
                        ) : (
                          <>
                            <Play className="w-4 h-4 mr-2" />
                            Başlat
                          </>
                        )}
                      </Button>
                    )}
                  </div>
                </CardContent>
              </Card>
            </motion.div>
          ))}
        </div>
      ) : (
        <Card className="glass">
          <div className="divide-y divide-border">
            {filteredChannels.map((channel) => (
              <div
                key={channel.id}
                className={cn(
                  "flex items-center gap-4 p-4 hover:bg-secondary/50 transition-colors",
                  selectedChannels.has(channel.id) && "bg-primary/5"
                )}
              >
                <button
                  onClick={() => toggleChannelSelection(channel.id)}
                  type="button"
                >
                  {selectedChannels.has(channel.id) ? (
                    <CheckSquare className="w-4 h-4 text-primary" />
                  ) : (
                    <SquareIcon className="w-4 h-4 text-muted-foreground" />
                  )}
                </button>
                <div className={cn("status-dot", getStatusDotClass(channel.status))} />
                <div className="flex-1 min-w-0">
                  <p className="font-medium truncate">{channel.name}</p>
                  <p className="text-sm text-muted-foreground truncate">
                    {channel.source_url}
                  </p>
                </div>
                <span className={cn("text-sm capitalize", getStatusColor(channel.status))}>
                  {getStatusText(channel.status)}
                </span>
                {channel.status === "running" && (
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => window.open(getStreamUrl(channel), "_blank")}
                    title="Yayını İzle"
                    className="text-success hover:text-success"
                  >
                    <ExternalLink className="w-4 h-4" />
                  </Button>
                )}
                <div className="flex gap-1">
                  {channel.status === "running" ? (
                    <>
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => handleStop(channel.id)}
                        disabled={actionLoading === channel.id}
                      >
                        <Square className="w-4 h-4" />
                      </Button>
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => handleRestart(channel.id)}
                        disabled={actionLoading === channel.id}
                      >
                        <RotateCcw className="w-4 h-4" />
                      </Button>
                    </>
                  ) : (
                    <Button
                      variant="success"
                      size="sm"
                      onClick={() => handleStart(channel.id)}
                      disabled={actionLoading === channel.id}
                    >
                      <Play className="w-4 h-4" />
                    </Button>
                  )}
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => router.push(`/channels/${channel.id}/edit`)}
                    title="Düzenle"
                  >
                    <Edit className="w-4 h-4" />
                  </Button>
                  <Button
                    variant="ghost"
                    size="sm"
                    className="text-destructive hover:text-destructive"
                    onClick={() => handleDelete(channel.id)}
                    disabled={actionLoading === channel.id}
                  >
                    <Trash2 className="w-4 h-4" />
                  </Button>
                </div>
              </div>
            ))}
          </div>
        </Card>
      )}

      {/* Channel Dialog - Only for Creating New Channels */}
      <ChannelDialog
        open={dialogOpen}
        onOpenChange={setDialogOpen}
        channel={null}
        onSuccess={fetchChannels}
      />
    </div>
  );
}

