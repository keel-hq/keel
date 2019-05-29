import Vue from 'vue'
import Vuex from 'vuex'

import app from './modules/app'
import user from './modules/user'
import tracked from './modules/tracked'
import resources from './modules/resources'
import approvals from './modules/approvals'
import audit from './modules/audit'
import stats from './modules/stats'
import permission from './modules/permission'
import getters from './getters'

Vue.use(Vuex)

export default new Vuex.Store({
  modules: {
    app,
    user,
    permission,
    tracked,
    resources,
    approvals,
    audit,
    stats
  },
  state: {

  },
  mutations: {

  },
  actions: {

  },
  getters
})
