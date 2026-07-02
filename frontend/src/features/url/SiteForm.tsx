import { useState } from "react"
import { useForm, Controller, type Control } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import { z } from "zod"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Textarea } from "@/components/ui/textarea"
import { Loader2, Download } from "lucide-react"
import { URLService } from "../../../bindings/collections/internal/service"
import type { Site } from "../../../bindings/collections/internal/model"
import { TagInput } from "@/components/TagInput"
import { FormField } from "@/components/FormField"
import { toast } from "sonner"
import { FormActions } from "@/components/FormActions"
import { FileUpload } from "@/components/FileUpload"
import { useURLNormalize } from "./hooks"
import { callService } from "@/lib/async"
import { str, arr } from "@/lib/safe"

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
function siteDefaults(site?: Site | null) {
  return {
    title: str(site?.title),
    url: str(site?.url),
    description: str(site?.description),
    tags: arr(site?.tags),
    icon: str(site?.icon),
    attachments: arr(site?.attachments).map((a) => ({ filename: a.filename, path: a.filename })),
  }
}

export function SiteForm({ site, onSave, onCancel }: SiteFormProps) {
  const isEdit = !!site
  const defaults = siteDefaults(site)
  const [tags, setTags] = useState<string[]>(defaults.tags)
  const [fetching, setFetching] = useState(false)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState("")
  const [fetchedIcon, setFetchedIcon] = useState<string>(defaults.icon)

  const [attachments, setAttachments] = useState<{ filename: string; path: string }[]>(
    defaults.attachments,
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
      title: defaults.title,
      url: defaults.url,
      description: defaults.description,
    },
  })

  const urlValue = watch("url")
  const normalizedURL = useURLNormalize(urlValue)

  /** 抓取页面元数据，回填 title、description、icon */
  async function handleFetch() {
    const url = urlValue?.trim()
    if (!url) {
      setError("请先输入 URL")
      return
    }
    setFetching(true)
    setError("")
    const [meta, err] = await callService(() => URLService.FetchMetadata({ url }))
    setFetching(false)
    if (err) {
      setError("fetch failed: " + err)
      return
    }
    if (meta) {
      if (meta.title) setValue("title", meta.title, { shouldValidate: true })
      if (meta.description) setValue("description", meta.description)
      if (meta.icon) setFetchedIcon(meta.icon)
    } else {
      setError("no metadata found")
    }
  }

  const onSubmit = async (values: SiteFormValues) => {
    setSaving(true)
    setError("")
    const serviceFn =
      isEdit && site
        ? () =>
            URLService.UpdateSite({
              id: site.id,
              title: values.title,
              description: values.description || null,
              tags,
              icon: fetchedIcon || site.icon || undefined,
              attachments: buildAttachmentsForSubmit(site, attachments),
            })
        : () =>
            URLService.CreateSite({
              title: values.title,
              url: values.url,
              description: values.description || undefined,
              tags: tags.length > 0 ? tags : undefined,
              icon: fetchedIcon || undefined,
            })
    const [, err] = await callService(serviceFn)
    setSaving(false)
    if (err) {
      setError(err)
      toast.error(err)
      return
    }
    toast.success(isEdit ? "站点已更新" : "站点已创建")
    onSave()
  }

  return (
    <form
      onSubmit={handleSubmit(onSubmit)}
      className="flex flex-col gap-4 px-6 py-5 overflow-y-auto"
    >
      <h2 className="text-base font-semibold">{isEdit ? "编辑站点" : "新建站点"}</h2>

      <URLField
        control={control}
        isEdit={isEdit}
        fetching={fetching}
        onFetch={handleFetch}
        error={errors.url?.message}
        normalizedURL={normalizedURL}
      />

      {/* 标题 */}
      <FormField label="标题" required error={errors.title?.message}>
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
              placeholder="站点描述（可选）"
              rows={3}
            />
          )}
        />
      </FormField>

      {/* 标签 */}
      <FormField label="标签">
        <TagInput value={tags} onChange={setTags} />
      </FormField>

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
      {!isEdit && <p className="text-xs text-muted-foreground">附件可在站点创建后添加</p>}

      {/* 错误提示 */}
      {error && <p className="text-xs text-destructive">{error}</p>}

      {/* 操作按钮 */}
      <FormActions saving={saving} onCancel={onCancel} />
    </form>
  )
}

// ─── Internal ───────────────────────────────────────────────

/**
 * 根据前端 attachments 的当前顺序，从原始 site.attachments 重建完整 Attachment 列表。
 * 新上传的文件（原始列表中不存在）用 filename 生成最小 Attachment 对象。
 */
function buildAttachmentsForSubmit(
  site: Site,
  currentFiles: { filename: string; path: string }[],
) {
  const origMap = new Map(arr(site.attachments).map((a) => [a.filename, a]))
  return currentFiles.map((f) => {
    const orig = origMap.get(f.filename)
    if (orig) return orig
    return { filename: f.filename, label: "", size: 0, uploaded_at: "" }
  })
}

/** URL 输入字段：输入框 + 抓取按钮 + 错误提示 + 标准化预览 */
function URLField({
  control,
  isEdit,
  fetching,
  onFetch,
  error,
  normalizedURL,
}: {
  control: Control<SiteFormValues>
  isEdit: boolean
  fetching: boolean
  onFetch: () => void
  error?: string
  normalizedURL: string
}) {
  return (
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
            onClick={onFetch}
            disabled={fetching}
          >
            {fetching
              ? <Loader2 size={14} className="animate-spin" />
              : <Download size={14} />}
            抓取
          </Button>
        )}
      </div>
      {error && <p className="text-xs text-destructive">{error}</p>}
      {normalizedURL && (
        <p className="text-xs text-muted-foreground">
          标准化：<span className="font-mono">{normalizedURL}</span>
        </p>
      )}
    </div>
  )
}
