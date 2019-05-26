import Vue from 'vue'
import api from '@/api/index.js'
// import { getInfo, logout } from '@/api/login'
import { ACCESS_TOKEN, USERNAME, PASSWORD } from '@/store/mutation-types'
import { welcome } from '@/utils/util'

const user = {
  state: {
    token: '',
    name: '',
    welcome: '',
    avatar: '',
    roles: [],
    info: {},
    credentials: {},
    error: null
  },

  mutations: {
    SET_TOKEN: (state, token) => {
      state.token = token
    },
    SET_NAME: (state, { name, welcome }) => {
      state.name = name
      state.welcome = welcome
    },
    SET_AVATAR: (state, avatar) => {
      state.avatar = avatar
    },
    SET_ROLES: (state, roles) => {
      state.roles = roles
    },
    SET_INFO: (state, info) => {
      state.info = info
    },
    SET_CREDENTIALS: (state, credentials) => {
      state.credentials = credentials
    },
    SET_ERROR: (state, error) => {
      state.error = error
    }
  },

  actions: {
    LoginSuccess ({ commit }, userInfo) {
      return new Promise((resolve, reject) => {
        Vue.ls.set(USERNAME, userInfo.username, 7 * 24 * 60 * 60 * 1000)
        Vue.ls.set(PASSWORD, userInfo.password, 7 * 24 * 60 * 60 * 1000)

        commit('SET_TOKEN', userInfo.username)
        Vue.ls.set(ACCESS_TOKEN, userInfo.username, 7 * 24 * 60 * 60 * 1000)

        commit('SET_CREDENTIALS', userInfo)
        resolve()
      })
    },

    // User info
    GetInfo ({ commit }) {
      // return api.get('user/info')
      return api.get('auth/user')
        .then((response) => {
          commit('SET_INFO', response)
          commit('SET_ROLES', response.role_id)
          commit('SET_NAME', { name: response.name, welcome: welcome() })
        })
        .catch((error) => {
          if (error.status === 401) {
            commit('SET_CREDENTIALS', {})
            commit('SET_TOKEN', '')
            Vue.ls.remove(ACCESS_TOKEN)
            Vue.ls.remove(USERNAME)
            Vue.ls.remove(PASSWORD)
          }

          commit('SET_ERROR', error)
        })
    },

    LogoutSuccess ({ commit }) {
      return new Promise((resolve, reject) => {
        commit('SET_CREDENTIALS', {})
        commit('SET_TOKEN', '')
        Vue.ls.remove(ACCESS_TOKEN)
        Vue.ls.remove(USERNAME)
        Vue.ls.remove(PASSWORD)
        resolve()
      })
    }
  }
}

export default user
