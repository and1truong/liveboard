export function EmptyState({
  title,
  detail,
}: {
  title: string
  detail?: string
}): JSX.Element {
  return (
    <div className="flex h-full items-center justify-center p-8 text-center">
      <div>
        <p className="text-base font-medium text-slate-700 dark:text-slate-300">{title}</p>
        {detail && <p className="mt-1 text-sm text-slate-500 dark:text-slate-400">{detail}</p>}
      </div>
    </div>
  )
}
