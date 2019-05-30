import { asyncRouterMap, constantRouterMap } from '@/config/router.config'

const permission = {
  state: {
    routers: constantRouterMap,
    addRouters: asyncRouterMap
  },
  mutations: {
    SET_ROUTERS: (state, routers) => {
      state.addRouters = routers
      state.routers = constantRouterMap.concat(routers)
    }
  },
  actions: {
    GenerateRoutes ({ commit }, data) {
      return new Promise(resolve => {
        // const { roles } = data
        console.log('GenerateRoutes - not doing anything')
        // console.log('generating routes')
        // console.log(roles)
        // const accessedRouters = filterAsyncRouter(asyncRouterMap, roles)
        // commit('SET_ROUTERS', accessedRouters)
        resolve()
      })
    }
  }
}

export default permission
