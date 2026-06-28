import { useState, useEffect } from "react"
import { useForm } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import { z } from "zod"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Loader2, AlertCircle, CheckCircle2 } from "lucide-react"
import { URLService } from "../../../bindings/collections/internal/service"
import type { Bookmark } from "../../../bindings/collections/internal/model"
import { TagInput } from "@/components/TagInput"

const bookmarkSchema = z.object({
  url: z.string().url("请输入有效的 URL"),
  title: z.string().min(1, "标题不能为空"),
  description: z.string().optional(),
})

type BookmarkFormValues = z.infer<typeof bookmarkSchema>

interface BookmarkFormProps {
  /** 编辑模式传入已有书签，创建模式不传 */
  bookmark?: Bookmark | null
  onSave: () => void
  onCancel: () => void
}

/**
 * 书签表单：创建和编辑共用。
 * 创建时输入 URL 后自动调用 LookupSiteByURL 检查域名是否有对应站点。
 * URL 一旦创建不可修改。
 */
export function BookmarkForm({ bookmark, onSave, onCancel }: BookmarkFormProps) {
  const isEdit = !!bookmark
  const [tags, setTags] = useState<string[]>(bookmark?.tags ?? [])
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState("")

  // 站点匹配状态（仅创建模式）
  const [lookupState, setLookupState] = useState<
    | { status: "idle" }
    | { status: "checking" }
    | { status: "found"; domain: string }
    | { status: "not_found"; domain: string }
  >({ status: "idle" })

  const {
    register,
    handleSubmit,
    watch,
    formState: { errors },
  } = useForm<BookmarkFormValues>({
    resolver: zodResolver(bookmarkSchema),
    defaultValues: {
      url: bookmark?.url ?? "",
      title: bookmark?.title ?? "",
      description: bookmark?.description ?? "",
    },
  })

  // 创建模式下，URL 变化时检查对应站点
  const urlValue = watch("url")
  useEffect(() => {
    if (isEdit || !urlValue) {
      setLookupState({ status: "idle" })
      return
    }
    // 简单校验是否像 URL
    try { new URL(urlValue) } catch { return }

    const timer = setTimeout(async () => {
      setLookupState({ status: "checking" })
      try {
        const result = await URLService.LookupSiteByURL({ url: urlValue })
        if (result?.found) {
          setLookupState({ status: "found", domain: result.domain })
        } else {
          setLookupState({ status: "not_found", domain: result?.domain ?? "" })
        }
      } catch {
        setLookupState({ status: "idle" })
      }
    }, 500)

    return () => clearTimeout(timer)
  }, [urlValue, isEdit])

  const onSubmit = async (values: BookmarkFormValues) => {
    setSaving(true)
    setError("")
    try {
      if (isEdit && bookmark) {
        await URLService.UpdateBookmark({
          id: bookmark.id,
          title: values.title,
          description: values.description ?? null,
          tags: tags,
        })
      } else {
        await URLService.CreateBookmark({
          url: values.url,
          title: values.title,
          description: values.description,
          tags: tags.length > 0 ? tags : undefined,
        })
      }
      onSave()
    } catch (e: any) {
      setError(e?.message ?? "保存失败")
    } finally {
      setSaving(false)
    }
  }

  return (
    <form onSubmit={handleSubmit(onSubmit)} className="flex flex-col gap-4 px-6 py-5">
      <h2 className="text-base font-semibold">
        {isEdit ? "编辑书签" : "新建书签"}
      </h2>

      {/* URL */}
      <div className="flex flex-col gap-1">
        <label className="text-sm">
          URL <span className="text-destructive">*</span>
        </label>
        <Input
          {...register("url")}
          placeholder="https://example.com/page"
          disabled={isEdit}
          className="h-8 text-sm"
        />
        {errors.url && <p className="text-xs text-destructive">{errors.url.message}</p>}

        {/* 站点匹配提示 */}
        {!isEdit && lookupState.status === "checking" && (
          <p className="text-xs text-muted-foreground flex items-center gap-1">
            <Loader2 size={12} className="animate-spin" /> 检查站点...
          </p>
        )}
        {!isEdit && lookupState.status === "found" && (
          <p className="text-xs text-muted-foreground flex items-center gap-1">
            <CheckCircle2 size={12} className="text-primary" />
            已匹配站点（{lookupState.domain}）
          </p>
        )}
        {!isEdit && lookupState.status === "not_found" && (
          <p className="text-xs text-destructive flex items-center gap-1">
            <AlertCircle size={12} />
            无对应站点（{lookupState.domain}），书签将存入临时队列
          </p>
        )}
      </div>

      {/* 标题 */}
      <div className="flex flex-col gap-1">
        <label className="text-sm">
          标题 <span className="text-destructive">*</span>
        </label>
        <Input {...register("title")} placeholder="书签标题" className="h-8 text-sm" />
        {errors.title && <p className="text-xs text-destructive">{errors.title.message}</p>}
      </div>

      {/* 描述 */}
      <div className="flex flex-col gap-1">
        <label className="text-sm">描述</label>
        <textarea
          {...register("description")}
          placeholder="书签描述（可选）"
          rows={3}
          className="flex w-full rounded-md border border-input bg-background px-3 py-2 text-sm placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring resize-none"
        />
      </div>

      {/* 标签 */}
      <div className="flex flex-col gap-1">
        <label className="text-sm">标签</label>
        <TagInput value={tags} onChange={setTags} />
      </div>

      {error && <p className="text-xs text-destructive">{error}</p>}

      <div className="flex justify-end gap-2 pt-2">
        <Button type="button" variant="ghost" size="sm" onClick={onCancel} disabled={saving}>
          取消
        </Button>
        <Button type="submit" size="sm" disabled={saving}>
          {saving && <Loader2 size={14} className="animate-spin mr-1" />}
          保存
        </Button>
      </div>
    </form>
  )
}
