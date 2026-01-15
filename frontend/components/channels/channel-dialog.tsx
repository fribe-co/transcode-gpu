"use client";

import { useState, useEffect } from "react";
import { Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { api, Channel } from "@/lib/api";
import { useToast } from "@/hooks/use-toast";

interface ChannelDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  channel: Channel | null;
  onSuccess: () => void;
}

export function ChannelDialog({
  open,
  onOpenChange,
  channel,
  onSuccess,
}: ChannelDialogProps) {
  const { toast } = useToast();
  const [loading, setLoading] = useState(false);
  const [name, setName] = useState("");
  const [sourceUrl, setSourceUrl] = useState("");
  const [bitrate, setBitrate] = useState("5000k");
  const [preset, setPreset] = useState("veryfast");

  useEffect(() => {
    if (!open) {
      setName("");
      setSourceUrl("");
      setBitrate("5000k");
      setPreset("veryfast");
    }
  }, [open]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);

    const data = {
      name,
      source_url: sourceUrl,
      output_config: {
        codec: "libx264",
        bitrate,
        resolution: "1920x1080",
        preset,
        profile: "high",
      },
    };

    const result = await api.createChannel(data);

    if (result.error) {
      toast({ title: "Hata", description: result.error, variant: "destructive" });
    } else {
      toast({
        title: "Başarılı",
        description: "Kanal oluşturuldu. Düzenlemek için kanal kartındaki düzenle butonuna tıklayın.",
      });
      onOpenChange(false);
      onSuccess();
    }

    setLoading(false);
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[500px] glass">
        <form onSubmit={handleSubmit}>
          <DialogHeader>
            <DialogTitle>Yeni Kanal Oluştur</DialogTitle>
            <DialogDescription>
              Yeni bir video transcoding kanalı ekleyin. Logo ve detaylı ayarlar için kanal oluşturulduktan sonra düzenle butonunu kullanın.
            </DialogDescription>
          </DialogHeader>
          <div className="grid gap-4 py-4">
            <div className="space-y-2">
              <Label htmlFor="name">Kanal Adı</Label>
              <Input
                id="name"
                placeholder="Kanalım"
                value={name}
                onChange={(e) => setName(e.target.value)}
                required
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="source">Kaynak URL</Label>
              <Input
                id="source"
                placeholder="rtmp://example.com/live/stream"
                value={sourceUrl}
                onChange={(e) => setSourceUrl(e.target.value)}
                required
              />
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label htmlFor="bitrate">Bitrate</Label>
                <Input
                  id="bitrate"
                  placeholder="5000k"
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
          </div>
          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={() => onOpenChange(false)}
            >
              İptal
            </Button>
            <Button type="submit" disabled={loading}>
              {loading ? (
                <>
                  <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                  Oluşturuluyor...
                </>
              ) : (
                "Kanal Oluştur"
              )}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
