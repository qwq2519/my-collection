import { Button } from "@/components/ui/button"
import { Loader2 } from "lucide-react"

interface FormActionsProps {
  saving: boolean
  onCancel: () => void
  submitLabel?: string
}

/**
 * 表单操作按钮组：取消 + 提交（带 loading 状态）。
 * 统一右对齐布局，所有表单复用。
 */
export function FormActions({ saving, onCancel, submitLabel = "保存" }: FormActionsProps) {
  return (
    <div className="flex justify-end gap-2 pt-2">
      <Button type="button" variant="ghost" size="sm" onClick={onCancel} disabled={saving}>
        取消
      </Button>
      <Button type="submit" size="sm" disabled={saving}>
        {saving && <Loader2 size={14} className="animate-spin mr-1" />}
        {submitLabel}
      </Button>
    </div>
  )
}
