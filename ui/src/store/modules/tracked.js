import api from '@/api/index.js'

const tracked = {
  state: {
    images: [],
    error: null
  },

  mutations: {
    SET_IMAGES: (state, images) => {
      var arrayLength = images.length
      // adding IDs so that the table is happy
      for (var i = 0; i < arrayLength; i++) {
        images[i].id = i.toString()
        if (images[i].trigger === 'default') {
          images[i].trigger = 'webhook/GCR'
        }
      }
      state.images = images
    },
    SET_ERROR: (state, error) => {
      state.error = error
    }
  },

  actions: {
    GetTrackedImages ({ commit }) {
      return api.get('tracked')
        .then((response) => {
          commit('SET_IMAGES', response)
        })
        .catch((error) => commit('SET_ERROR', error))
    },
    SetTracking ({ commit }, payload) {
      commit('SET_ERROR', null)
      return api.put(`tracked`, payload)
        .then((response) => commit('SET_ERROR', null))
        .catch((error) => commit('SET_ERROR', error))
    }
  }
}

export default tracked
