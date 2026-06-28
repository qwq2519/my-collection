import { useState } from "react"
import { useForm } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import { z } from "zod"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Loader2, Download } from "lucide-react"
import { URLService } from "../../../bindings/collections/internal/service"
import type { Site } from "../../../bindings/collections/internal/model"
import { TagInput } from "@/components/TagInput"

const siteSchema = z.object({
  title: z.string().min(1, "标题不能为空"),
  url: z.string().url("请输入有效的 URL"),
  description: z.string().optional(),
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
 * 包含"抓取"按钮调用 FetchMetadata 回填 title/description。
 */
export function SiteForm({ site, onSave, onCancel }: SiteFormProps) {
  const isEdit = !!site
  const [tags, setTags] = useState<string[]>(site?.tags ?? [])
  const [fetching, setFetching] = useState(false)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState("")

  const {
    register,
    handleSubmit,
    setValue,
    getValues,
    formState: { errors },
  } = useForm<SiteFormValues>({
    resolver: zodResolver(siteSchema),
    defaultValues: {
      title: site?.title ?? "",
      url: site?.url ?? "",
      description: site?.description ?? "",
    },
  })

  /** 抓取页面元数据，回填 title 和 description */
  const handleFetch = async () => {
    const url = getValues("url")
    if (!url) return
    setFetching(true)
    setError("")
    try {
      const meta = await URLService.FetchMetadata({ url })
      if (meta?.title && !getValues("title")) {
        setValue("title", meta.title)
      }
      if (meta?.description && !getValues("description")) {
        setValue("description", meta.description)
      }
    } catch (e: any) {
      setError("抓取失败：" + (e?.message ?? "网络异常"))
    } finally {
      setFetching(false)
    }
  }

  const onSubmit = async (values: SiteFormValues) => {
    setSaving(true)
    setError("")
    try {
      if (isEdit && site) {
        await URLService.UpdateSite({
          id: site.id,
          title: values.title,
          description: values.description ?? null,
          tags: tags,
        })
      } else {
        await URLService.CreateSite({
          title: values.title,
          url: values.url,
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
        {isEdit ? "编辑站点" : "新建站点"}
      </h2>

      {/* URL + 抓取 */}
      <div className="flex flex-col gap-1">
        <label className="text-sm">
          URL <span className="text-destructive">*</span>
        </label>
        <div className="flex gap-2">
          <Input
            {...register("url")}
            placeholder="https://example.com"
            disabled={isEdit}
            className="flex-1 h-8 text-sm"
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
      </div>

      {/* 标题 */}
      <div className="flex flex-col gap-1">
        <label className="text-sm">
          标题 <span className="text-destructive">*</span>
        </label>
        <Input {...register("title")} placeholder="站点名称" className="h-8 text-sm" />
        {errors.title && <p className="text-xs text-destructive">{errors.title.message}</p>}
      </div>

      {/* 描述 */}
      <div className="flex flex-col gap-1">
        <label className="text-sm">描述</label>
        <textarea
          {...register("description")}
          placeholder="站点描述（可选）"
          rows={3}
          className="flex w-full rounded-md border border-input bg-background px-3 py-2 text-sm placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring resize-none"
        />
      </div>

      {/* 标签 */}
      <div className="flex flex-col gap-1">
        <label className="text-sm">标签</label>
        <TagInput value={tags} onChange={setTags} />
      </div>

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
