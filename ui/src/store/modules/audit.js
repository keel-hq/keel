import api from '@/api/index.js'

const audit = {
  state: {
    audit_logs: [],
    pagination: {
      limit: 100,
      offset: 0,
      total: 5
    },
    error: null,
    loading: false
  },

  mutations: {
    SET_AUDIT_LOGS: (state, logs) => {
      state.audit_logs = logs
    },
    SET_PAGINATION: (state, pagination) => {
      state.pagination = pagination
    },
    SET_ERROR: (state, error) => {
      state.error = error
    },
    SET_LOADING: (state, loading) => {
      state.loading = loading
    }
  },

  actions: {
    GetAuditLogs ({ commit }, query) {
      commit('SET_ERROR', null)
      commit('SET_LOADING', true)
      return api.get(`audit?filter=${query.filter}&limit=${query.limit}&offset=${query.offset}`)
        .then((response) => {
          commit('SET_AUDIT_LOGS', response.data)
          const pagination = {
            limit: response.limit,
            offset: response.offset,
            total: response.total
          }
          commit('SET_PAGINATION', pagination)
          commit('SET_LOADING', false)
        })
        .catch((error) => {
          commit('SET_LOADING', false)
          commit('SET_ERROR', error)
        })
    }
  }
}

export default audit
