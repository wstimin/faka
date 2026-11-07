export interface Delivery {
  status?: string
  type?: string
  payload?: string
  payload_line_count?: number
  delivery_data?: { note?: string; entries?: Array<{ key?: string; value?: string }> }
  logistics?: Delivery['delivery_data']
}

export interface StoreOrder {
  order_no: string
  status: string
  items?: any[]
  children?: StoreOrder[]
  fulfillment?: Delivery | null
  [key: string]: any
}

export const orderItems = (order: StoreOrder): any[] =>
  order.items?.length ? order.items : (order.children || []).flatMap(child => child.items || [])

export const orderPaymentState = (order: StoreOrder): string => {
  if (['refunded', 'partially_refunded'].includes(order.status)) return order.status
  if (['paid', 'fulfilling', 'partially_delivered', 'delivered', 'completed'].includes(order.status)) return 'paid'
  if (['pending_payment', 'canceled', 'expired'].includes(order.status)) return 'unpaid'
  return 'unknown'
}

export const orderShipmentState = (order: StoreOrder): string => {
  const state = deliveryState(order, true)
  if (state === 'ready') return 'delivered'
  if (state === 'partial') return 'partial'
  if (['paid', 'fulfilling'].includes(order.status)) return 'waiting'
  if (['pending_payment', 'canceled', 'expired'].includes(order.status)) return 'notShipped'
  return 'unknown'
}

export const deliveryGroups = (order: StoreOrder): StoreOrder[] => {
  const groups = order.children?.length ? [...order.children] : []
  if (!groups.length || order.fulfillment) groups.push(order)
  return groups
}

export const deliveryText = (delivery?: Delivery | null): string => {
  const data = delivery?.delivery_data || delivery?.logistics
  if (data) {
    const lines = [data.note || '', ...(data.entries || []).map(entry =>
      [entry.key, entry.value].filter(Boolean).join(': '),
    )].filter(Boolean)
    if (lines.length) return lines.join('\n')
  }
  return delivery?.payload || ''
}

export const isDeliveryTruncated = (delivery?: Delivery | null): boolean =>
  Number(delivery?.payload_line_count || 0) > 100

export const deliveredGroups = (order: StoreOrder): StoreOrder[] =>
  deliveryGroups(order).filter(group => group.fulfillment?.status === 'delivered' && deliveryText(group.fulfillment))

// 不按分隔符拆账号或卡密，保留商家原始交付格式；截断预览只允许下载完整内容。
export const copyableDeliveryText = (order: StoreOrder): string => {
  const groups = deliveredGroups(order)
  if (!groups.length || groups.some(group => isDeliveryTruncated(group.fulfillment))) return ''
  return groups.map(group => deliveryText(group.fulfillment)).join('\n\n')
}

export const deliveryState = (order: StoreOrder, summary = false): string => {
  if (['pending_payment', 'canceled', 'expired', 'refunded', 'partially_refunded'].includes(order.status)) return order.status
  const groups = deliveryGroups(order)
  const count = groups.filter(group => group.fulfillment?.status === 'delivered').length
  if (count === groups.length && count > 0) return 'ready'
  if (count > 0 || order.status === 'partially_delivered') return 'partial'
  if (summary && ['completed', 'delivered'].includes(order.status)) return 'ready'
  return 'waiting'
}
