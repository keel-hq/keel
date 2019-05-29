import api from '@/api/index.js'

const stats = {
  state: {
    stats: [],
    totalUpdatesThisPeriod: 0,
    error: null,
    loading: false
  },

  mutations: {
    SET_STATS: (state, stats) => {
      state.stats = stats

      // calculate stats
      let total = 0
      var arrayLength = stats.length
      for (var i = 0; i < arrayLength; i++) {
        total += stats[i].updates
      }
      state.totalUpdatesThisPeriod = total
    },
    SET_ERROR: (state, error) => {
      state.error = error
    },
    SET_LOADING: (state, loading) => {
      state.loading = loading
    }
  },

  actions: {
    GetStats ({ commit }) {
      commit('SET_ERROR', null)
      commit('SET_LOADING', true)
      return api.get(`stats`)
        .then((response) => {
          commit('SET_STATS', response)
          commit('SET_LOADING', false)
        })
        .catch((error) => {
          commit('SET_LOADING', false)
          commit('SET_ERROR', error)
        })
    }
  }
}

export default stats
