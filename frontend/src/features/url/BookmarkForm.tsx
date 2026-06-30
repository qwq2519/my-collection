import { useState } from "react"
import { useForm, Controller } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import { z } from "zod"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Textarea } from "@/components/ui/textarea"
import { Loader2, AlertCircle, CheckCircle2 } from "lucide-react"
import { URLService } from "../../../bindings/collections/internal/service"
import type { Bookmark } from "../../../bindings/collections/internal/model"
import { TagInput } from "@/components/TagInput"
import { FormField } from "@/components/FormField"
import { toast } from "sonner"
import { useSiteLookup } from "./hooks"
import { callService } from "@/lib/async"
import { str, arr } from "@/lib/safe"

const bookmarkSchema = z.object({
  url: z.string().min(1, "请输入 URL"),
  title: z.string().min(1, "标题不能为空"),
  description: z.string(),
})

type BookmarkFormValues = z.infer<typeof bookmarkSchema>

interface BookmarkFormProps {
  /** 编辑模式传入已有书签，创建模式不传 */
  bookmark?: Bookmark | null
  onSave: () => void
  onCancel: () => void
  /** 当域名无对应站点时，提供快速切换到创建站点的入口 */
  onCreateSite?: () => void
}

/**
 * 书签表单：创建和编辑共用。
 * 创建时输入 URL 后自动调用 LookupSiteByURL 检查域名是否有对应站点，
 * 并实时展示标准化后的 URL。
 */
function bookmarkDefaults(bm?: Bookmark | null) {
  return {
    url: str(bm?.url),
    title: str(bm?.title),
    description: str(bm?.description),
    tags: arr(bm?.tags),
  }
}

export function BookmarkForm({ bookmark, onSave, onCancel, onCreateSite }: BookmarkFormProps) {
  const isEdit = !!bookmark
  const defaults = bookmarkDefaults(bookmark)
  const [tags, setTags] = useState<string[]>(defaults.tags)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState("")

  const {
    control,
    handleSubmit,
    watch,
    formState: { errors },
  } = useForm<BookmarkFormValues>({
    resolver: zodResolver(bookmarkSchema),
    defaultValues: {
      url: defaults.url,
      title: defaults.title,
      description: defaults.description,
    },
  })

  const urlValue = watch("url")
  const { normalizedURL, lookupState } = useSiteLookup(urlValue, !isEdit)

  const onSubmit = async (values: BookmarkFormValues) => {
    setSaving(true)
    setError("")
    const serviceFn =
      isEdit && bookmark
        ? () =>
            URLService.UpdateBookmark({
              id: bookmark.id,
              title: values.title,
              description: values.description || null,
              tags,
            })
        : () =>
            URLService.CreateBookmark({
              url: values.url,
              title: values.title,
              description: values.description || undefined,
              tags: tags.length > 0 ? tags : undefined,
            })
    const [, err] = await callService(serviceFn)
    setSaving(false)
    if (err) {
      setError(err)
      toast.error(err)
      return
    }
    toast.success(isEdit ? "书签已更新" : "书签已创建")
    onSave()
  }

  return (
    <form onSubmit={handleSubmit(onSubmit)} className="flex flex-col gap-4 px-6 py-5">
      <h2 className="text-base font-semibold">{isEdit ? "编辑书签" : "新建书签"}</h2>

      {/* URL */}
      <div className="flex flex-col gap-1">
        <label className="text-sm">
          URL <span className="text-destructive">*</span>
        </label>
        <Controller
          name="url"
          control={control}
          render={({ field }) => (
            <Input
              {...field}
              value={field.value ?? ""}
              placeholder="https://example.com/page"
              disabled={isEdit}
              className="h-8 text-sm"
            />
          )}
        />
        {errors.url && <p className="text-xs text-destructive">{errors.url.message}</p>}

        {!isEdit && (
          <SiteLookupHint
            normalizedURL={normalizedURL}
            lookupState={lookupState}
            onCreateSite={onCreateSite}
          />
        )}
      </div>

      {/* 标题 */}
      <FormField label="标题" required error={errors.title?.message}>
        <Controller
          name="title"
          control={control}
          render={({ field }) => (
            <Input
              {...field}
              value={field.value ?? ""}
              placeholder="书签标题"
              className="h-8 text-sm"
            />
          )}
        />
      </FormField>

      {/* 描述 */}
      <FormField label="描述">
        <Controller
          name="description"
          control={control}
          render={({ field }) => (
            <Textarea
              {...field}
              value={field.value ?? ""}
              placeholder="书签描述（可选）"
              rows={3}
            />
          )}
        />
      </FormField>

      {/* 标签 */}
      <FormField label="标签">
        <TagInput value={tags} onChange={setTags} />
      </FormField>

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

// ─── Internal ───────────────────────────────────────────────

/** URL 输入后的站点查找状态提示：标准化结果 + 站点匹配/未匹配反馈 */
function SiteLookupHint({
  normalizedURL,
  lookupState,
  onCreateSite,
}: {
  normalizedURL: string
  lookupState: import("./hooks").SiteLookupState
  onCreateSite?: () => void
}) {
  return (
    <>
      {normalizedURL && (
        <p className="text-xs text-muted-foreground">
          标准化：<span className="font-mono">{normalizedURL}</span>
        </p>
      )}
      {lookupState.status === "checking" && (
        <p className="text-xs text-muted-foreground flex items-center gap-1">
          <Loader2 size={12} className="animate-spin" /> 检查站点...
        </p>
      )}
      {lookupState.status === "found" && (
        <p className="text-xs text-muted-foreground flex items-center gap-1">
          <CheckCircle2 size={12} className="text-primary" />
          已匹配站点（{lookupState.domain}）
        </p>
      )}
      {lookupState.status === "not_found" && (
        <div className="flex flex-col gap-1">
          <p className="text-xs text-destructive flex items-center gap-1">
            <AlertCircle size={12} />
            无对应站点（{lookupState.domain}）
          </p>
          {onCreateSite && (
            <button
              type="button"
              onClick={onCreateSite}
              className="text-xs text-primary hover:underline self-start"
            >
              去创建站点 →
            </button>
          )}
        </div>
      )}
    </>
  )
}
