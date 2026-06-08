import { Card, CardHeader, CardTitle, CardContent } from '@/components/ui/card'
import { ScrollArea } from '@/components/ui/scroll-area'

interface JsonViewerProps {
  title: string
  data: unknown
  maxHeight?: string
  className?: string
  highlight?: boolean
}

export function JsonViewer({ title, data, maxHeight = '320px', className, highlight }: JsonViewerProps) {
  const jsonString = JSON.stringify(data, null, 2)

  return (
    <Card className={className}>
      <CardHeader className="py-3 px-4 flex-row items-center justify-between space-y-0">
        <CardTitle className="text-sm font-medium">{title}</CardTitle>
      </CardHeader>
      <CardContent className="p-0">
        <ScrollArea style={{ maxHeight }} className="border-t">
          <pre className="json-view p-4 text-muted-foreground whitespace-pre-wrap break-all">
            {jsonString}
          </pre>
        </ScrollArea>
      </CardContent>
    </Card>
  )
}
