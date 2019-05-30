// eslint-disable-next-line
import { UserLayout, BasicLayout, RouteView, BlankLayout, PageView } from '@/layouts'
import { bxAnaalyse } from '@/core/icons'

export const asyncRouterMap = [

  {
    path: '/',
    name: 'index',
    component: BasicLayout,
    meta: { title: 'Dashboard', auth: true },
    redirect: '/dashboard',
    children: [
      // dashboard
      {
        path: '/dashboard',
        name: 'dashboard',
        hideChildrenInMenu: true,
        component: () => import('@/views/dashboard/Analysis'),
        meta: { title: 'Dashboard', keepAlive: true, icon: bxAnaalyse, permission: [ 'dashboard' ], auth: true }
      },

      {
        path: '/tracked-images',
        name: 'tracked',
        hideChildrenInMenu: true,
        meta: { title: 'Tacked Images', keepAlive: true, icon: 'reconciliation', permission: [ 'dashboard' ], auth: true },
        component: () => import('@/views/tracked/TrackedImageList')
      },

      {
        path: '/approvals',
        name: 'approvals',
        hideChildrenInMenu: true,
        meta: { title: 'Approvals', keepAlive: true, icon: 'form', permission: [ 'dashboard' ], auth: true },
        component: () => import('@/views/approvals/Approvals')
      },

      {
        path: '/audit-logs',
        name: 'audit',
        component: () => import('@/views/audit/AuditLogs'),
        hideChildrenInMenu: true,
        meta: { title: 'Audit', keepAlive: true, icon: 'profile', permission: [ 'dashboard' ], auth: true }
      }
    ]
  },
  {
    path: '*', redirect: '/404', hidden: true
  }
]

/**
 *
 * @type { *[] }
 */
export const constantRouterMap = [

  // {
  //   path: '/',
  //   name: 'index',
  //   component: BasicLayout,
  //   meta: { title: 'Dashboard', auth: true },
  //   redirect: '/dashboard',
  //   children: [
  //     // dashboard
  //     {
  //       path: '/dashboard',
  //       name: 'dashboard',
  //       hideChildrenInMenu: true,
  //       component: () => import('@/views/dashboard/Analysis'),
  //       meta: { title: 'Dashboard', keepAlive: true, icon: bxAnaalyse, permission: [ 'dashboard' ], auth: true }
  //     },

  //     {
  //       path: '/tracked-images',
  //       name: 'tracked',
  //       hideChildrenInMenu: true,
  //       meta: { title: 'Tacked Images', keepAlive: true, icon: 'reconciliation', permission: [ 'dashboard' ], auth: true },
  //       component: () => import('@/views/tracked/TrackedImageList')
  //     },

  //     {
  //       path: '/approvals',
  //       name: 'approvals',
  //       hideChildrenInMenu: true,
  //       meta: { title: 'Approvals', keepAlive: true, icon: 'form', permission: [ 'dashboard' ], auth: true },
  //       component: () => import('@/views/approvals/Approvals')
  //     },

  //     {
  //       path: '/audit-logs',
  //       name: 'audit',
  //       component: () => import('@/views/audit/AuditLogs'),
  //       hideChildrenInMenu: true,
  //       meta: { title: 'Audit', keepAlive: true, icon: 'profile', permission: [ 'dashboard' ], auth: true }
  //     }
  //   ]
  // },

  {
    path: '/user',
    component: UserLayout,
    redirect: '/user/login',
    hidden: true,
    children: [
      {
        path: 'login',
        name: 'login',
        component: () => import(/* webpackChunkName: "user" */ '@/views/user/Login'),
        meta: { auth: false }
      }
    ]
  },

  {
    path: '/404',
    component: () => import(/* webpackChunkName: "fail" */ '@/views/exception/404'),
    meta: { auth: false }
  }

]
