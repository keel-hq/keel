import Vue from 'vue'

export default {
  get (url, request) {
    return Vue.http.get(url, request)
      .then((response) => Promise.resolve(response.body))
      .catch((error) => Promise.reject(error))
  },
  post (url, request) {
    return Vue.http.post(url, request)
      .then((response) => Promise.resolve(response))
      .catch((error) => Promise.reject(error))
  },
  put (url, request) {
    return Vue.http.put(url, request)
      .then((response) => Promise.resolve(response))
      .catch((error) => Promise.reject(error))
  },
  patch (url, request) {
    return Vue.http.patch(url, request)
      .then((response) => Promise.resolve(response))
      .catch((error) => Promise.reject(error))
  },
  delete (url, request) {
    return Vue.http.delete(url, request)
      .then((response) => Promise.resolve(response))
      .catch((error) => Promise.reject(error))
  }
}

// import { axios } from '@/utils/request'

// export default {
//   get (url, request) {
//     return axios({
//       url: url,
//       method: 'get',
//       data: request
//     })
//   },
//   put (url, request) {
//     return axios({
//       url: url,
//       method: 'put',
//       data: request
//     })
//   },
//   post (url, request) {
//     return axios({
//       url: url,
//       method: 'post',
//       data: request
//     })
//   },
//   delete (url, request) {
//     return axios({
//       url: url,
//       method: 'delete',
//       data: request
//     })
//   }
// }
