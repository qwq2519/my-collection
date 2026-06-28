import { useCallback, useState } from "react"
import { useForm, Controller } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import { z } from "zod"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Loader2, Download } from "lucide-react"
import { URLService } from "../../../bindings/collections/internal/service"
import type { Site } from "../../../bindings/collections/internal/model"
import { TagInput } from "@/components/TagInput"
import { extractError } from "@/lib/utils"
import { toast } from "sonner"
import { FileUpload } from "@/components/FileUpload"
import { useURLNormalize } from "./hooks"

const siteSchema = z.object({
  title: z.string().min(1, "标题不能为空"),
  url: z.string().min(1, "请输入 URL"),
  description: z.string(),
})

type SiteFormValues = z.infer<typeof siteSchema>

interface SiteFormProps {
  /** 编辑模式传入已有站点，创建模式不传 */
  site?: Site | null
  onSave: () => void
  onCancel: () => void
}

/**
 * 站点表单：创建和编辑共用。
 * - 输入 URL 后实时展示标准化结果
 * - "抓取"按钮调用 FetchMetadata 回填 title/description/icon
 * - 编辑模式支持附件上传
 */
export function SiteForm({ site, onSave, onCancel }: SiteFormProps) {
  const isEdit = !!site
  const [tags, setTags] = useState<string[]>(site?.tags ?? [])
  const [fetching, setFetching] = useState(false)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState("")
  const [fetchedIcon, setFetchedIcon] = useState<string>(site?.icon ?? "")

  // 附件（编辑模式）
  const [attachments, setAttachments] = useState<{ filename: string; path: string }[]>(
    () => (site?.attachments ?? []).map((a) => ({ filename: a.filename, path: a.filename }))
  )

  const {
    control,
    handleSubmit,
    setValue,
    watch,
    formState: { errors },
  } = useForm<SiteFormValues>({
    resolver: zodResolver(siteSchema),
    defaultValues: {
      title: site?.title ?? "",
      url: site?.url ?? "",
      description: site?.description ?? "",
    },
  })

  const urlValue = watch("url")
  const normalizedURL = useURLNormalize(urlValue)

  /** 抓取页面元数据，回填 title、description、icon */
  const handleFetch = useCallback(async () => {
    const url = urlValue?.trim()
    if (!url) {
      setError("请先输入 URL")
      return
    }
    setFetching(true)
    setError("")
    try {
      const meta = await URLService.FetchMetadata({ url })
      if (meta) {
        if (meta.title) setValue("title", meta.title, { shouldValidate: true })
        if (meta.description) setValue("description", meta.description)
        if (meta.icon) setFetchedIcon(meta.icon)
      } else {
        setError("no metadata found")
      }
    } catch (e: any) {
      setError("fetch failed: " + extractError(e))
    } finally {
      setFetching(false)
    }
  }, [urlValue, setValue])

  const onSubmit = async (values: SiteFormValues) => {
    setSaving(true)
    setError("")
    try {
      if (isEdit && site) {
        await URLService.UpdateSite({
          id: site.id,
          title: values.title,
          description: values.description || null,
          tags,
          icon: fetchedIcon || site.icon || undefined,
        })
      } else {
        await URLService.CreateSite({
          title: values.title,
          url: values.url,
          description: values.description || undefined,
          tags: tags.length > 0 ? tags : undefined,
          icon: fetchedIcon || undefined,
        })
      }
      toast.success(isEdit ? "站点已更新" : "站点已创建")
      onSave()
    } catch (e: unknown) {
      const msg = extractError(e)
      setError(msg)
      toast.error(msg)
    } finally {
      setSaving(false)
    }
  }

  return (
    <form onSubmit={handleSubmit(onSubmit)} className="flex flex-col gap-4 px-6 py-5 overflow-y-auto">
      <h2 className="text-base font-semibold">
        {isEdit ? "编辑站点" : "新建站点"}
      </h2>

      {/* URL + 抓取 */}
      <div className="flex flex-col gap-1">
        <label className="text-sm">
          URL <span className="text-destructive">*</span>
        </label>
        <div className="flex gap-2">
          <Controller
            name="url"
            control={control}
            render={({ field }) => (
              <Input
                {...field}
                value={field.value ?? ""}
                placeholder="https://example.com"
                disabled={isEdit}
                className="flex-1 h-8 text-sm"
              />
            )}
          />
          {!isEdit && (
            <Button
              type="button"
              variant="outline"
              size="sm"
              className="h-8 gap-1 shrink-0"
              onClick={handleFetch}
              disabled={fetching}
            >
              {fetching ? <Loader2 size={14} className="animate-spin" /> : <Download size={14} />}
              抓取
            </Button>
          )}
        </div>
        {errors.url && <p className="text-xs text-destructive">{errors.url.message}</p>}
        {normalizedURL && (
          <p className="text-xs text-muted-foreground">
            标准化：<span className="font-mono">{normalizedURL}</span>
          </p>
        )}
      </div>

      {/* 标题 */}
      <div className="flex flex-col gap-1">
        <label className="text-sm">
          标题 <span className="text-destructive">*</span>
        </label>
        <Controller
          name="title"
          control={control}
          render={({ field }) => (
            <Input
              {...field}
              value={field.value ?? ""}
              placeholder="站点名称"
              className="h-8 text-sm"
            />
          )}
        />
        {errors.title && <p className="text-xs text-destructive">{errors.title.message}</p>}
      </div>

      {/* 描述 */}
      <div className="flex flex-col gap-1">
        <label className="text-sm">描述</label>
        <Controller
          name="description"
          control={control}
          render={({ field }) => (
            <textarea
              {...field}
              value={field.value ?? ""}
              placeholder="站点描述（可选）"
              rows={3}
              className="flex w-full rounded-md border border-input bg-background px-3 py-2 text-sm placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring resize-none"
            />
          )}
        />
      </div>

      {/* 标签 */}
      <div className="flex flex-col gap-1">
        <label className="text-sm">标签</label>
        <TagInput value={tags} onChange={setTags} />
      </div>

      {/* 抓取到的 icon 预览 */}
      {fetchedIcon && (
        <div className="flex items-center gap-2">
          <label className="text-sm text-muted-foreground">图标</label>
          <img
            src={`/persist/url-assets/icons/${fetchedIcon}`}
            alt="site icon"
            className="w-6 h-6 rounded"
          />
        </div>
      )}

      {/* 附件（编辑模式） */}
      {isEdit && site && (
        <div className="flex flex-col gap-1">
          <label className="text-sm">附件</label>
          <FileUpload
            scene="site-attachment"
            entityId={site.id}
            files={attachments}
            onChange={setAttachments}
          />
        </div>
      )}
      {!isEdit && (
        <p className="text-xs text-muted-foreground">附件可在站点创建后添加</p>
      )}

      {/* 错误提示 */}
      {error && <p className="text-xs text-destructive">{error}</p>}

      {/* 操作按钮 */}
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
