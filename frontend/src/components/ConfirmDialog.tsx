import { useState } from "react"
import {
  AlertDialog,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog"
import { Button } from "@/components/ui/button"
import { Loader2 } from "lucide-react"
import { extractError } from "@/lib/utils"
import { toast } from "sonner"

interface ConfirmDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  title: string
  description: string
  /** 确认按钮文字，默认"确认" */
  confirmLabel?: string
  /** 是否为危险操作（确认按钮变红），默认 true */
  destructive?: boolean
  /** 支持异步：成功后自动关闭，失败时展示错误并保持打开 */
  onConfirm: () => void | Promise<void>
}

/**
 * 通用确认弹窗，用于删除等需要二次确认的操作。
 * 支持异步 onConfirm：操作期间显示 loading，失败时展示错误信息并保持弹窗打开。
 */
export function ConfirmDialog({
  open,
  onOpenChange,
  title,
  description,
  confirmLabel = "确认",
  destructive = true,
  onConfirm,
}: ConfirmDialogProps) {
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState("")

  const handleConfirm = async () => {
    setLoading(true)
    setError("")
    try {
      await onConfirm()
      onOpenChange(false)
    } catch (e: unknown) {
      const msg = extractError(e)
      setError(msg)
      toast.error(msg)
    } finally {
      setLoading(false)
    }
  }

  const handleOpenChange = (next: boolean) => {
    if (loading) return
    if (!next) setError("")
    onOpenChange(next)
  }

  return (
    <AlertDialog open={open} onOpenChange={handleOpenChange}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>{title}</AlertDialogTitle>
          <AlertDialogDescription>{description}</AlertDialogDescription>
        </AlertDialogHeader>
        {error && <p className="text-sm text-destructive px-1">{error}</p>}
        <AlertDialogFooter>
          <AlertDialogCancel disabled={loading}>取消</AlertDialogCancel>
          <Button
            onClick={handleConfirm}
            disabled={loading}
            className={
              destructive
                ? "bg-destructive text-destructive-foreground hover:bg-destructive/90"
                : ""
            }
          >
            {loading && <Loader2 size={14} className="animate-spin mr-1" />}
            {confirmLabel}
          </Button>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}
