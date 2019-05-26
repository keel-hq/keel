import api from '@/api/index.js'

const approvals = {
  state: {
    approvals: [],
    error: null
  },

  mutations: {
    SET_APPROVALS: (state, approvals) => {
      state.approvals = []
      state.approvals = approvals
    },
    SET_ERROR: (state, error) => {
      state.error = error
    },
    SET_APPROVAL_LOADING: (state, identifier) => {
      var arrayLength = state.approvals.length
      for (var i = 0; i < arrayLength; i++) {
        if (state.approvals[i].identifier === identifier) {
          state.approvals[i]._loading = true
        }
      }
    }
  },

  actions: {
    GetApprovals ({ commit }) {
      commit('SET_ERROR', null)
      return api.get('approvals')
        .then((response) => {
          commit('SET_APPROVALS', response)
        })
        .catch((error) => commit('SET_ERROR', error))
    },
    UpdateApproval ({ commit }, payload) { // can reject/approve
      commit('SET_ERROR', null)
      commit('SET_APPROVAL_LOADING', payload.identifier)
      return api.post(`approvals`, payload)
        .then((response) => commit('SET_ERROR', null))
        .catch((error) => commit('SET_ERROR', error))
    },
    SetApproval ({ commit }, payload) { // can increase/decrease approvals count
      commit('SET_ERROR', null)
      commit('SET_APPROVAL_LOADING', payload.identifier)
      return api.put(`approvals`, payload)
        .then((response) => commit('SET_ERROR', null))
        .catch((error) => commit('SET_ERROR', error))
    }
  }
}

export default approvals
