/**
 * 控制台前端路由 path 的唯一真相源。
 *
 * App.tsx 的路由注册、列表页跳转、详情页返回都引用这里，避免"注册 `/oauth-client/:id`
 * 却跳到 `/oauthClient/:id`"这类字面量漂移——那会让点击名称进入空白页（无匹配路由）。
 *
 * 约定：path 与后端动态菜单的 path 一致（绝对路径 + kebab-case，见 seed 的菜单定义）。
 */

/** OAuth 客户端列表 */
export const OAUTH_CLIENT_LIST_PATH = '/oauth-client'

/** OAuth 客户端详情路由（静态注册，不进侧边栏菜单；`:id` 为控制台内部主键 applicationClientID） */
export const OAUTH_CLIENT_DETAIL_ROUTE = '/oauth-client/:id'

/** 拼接某个客户端的详情地址（`:id` 用内部主键 applicationClientID，非客户端编码） */
export function oauthClientDetailPath(applicationClientID: string): string {
  return `${OAUTH_CLIENT_LIST_PATH}/${applicationClientID}`
}
