const getters = {
  device: state => state.app.device,
  theme: state => state.app.theme,
  color: state => state.app.color,
  token: state => state.user.token,
  avatar: state => state.user.avatar,
  nickname: state => state.user.name,
  welcome: state => state.user.welcome,
  roles: state => state.user.roles,
  userInfo: state => state.user.info,
  addRouters: state => state.permission.addRouters,
  multiTab: state => state.app.multiTab,
  trackedImages: state => state.tracked.images,
  resourcesAll: state => state.resources.resources,
  resourcesManaged: state => {
    var filtered = []
    state.resources.resources.map(function (resource) {
      if (resource.policy !== 'nil policy') {
        filtered.push(resource)
      }
    })
    return filtered
  },
  approvalsPending: state => {
    var filtered = []
    state.approvals.approvals.map(function (approval) {
      if (!approval.rejected && !approval.archived && approval.votesReceived < approval.votesRequired) {
        filtered.push(approval)
      }
    })
    return filtered
  },
  approvalsApprovedCount: state => {
    var arrayLength = state.approvals.approvals.length
    const approvals = state.approvals.approvals
    let count = 0
    for (var i = 0; i < arrayLength; i++) {
      if (!approvals[i].rejected && approvals[i].votesReceived >= approvals[i].votesRequired) {
        count++
      }
    }
    return count
  },
  approvalsRejectedCount: state => {
    var arrayLength = state.approvals.approvals.length
    const approvals = state.approvals.approvals
    let count = 0
    for (var i = 0; i < arrayLength; i++) {
      if (approvals[i].rejected) {
        count++
      }
    }
    return count
  },
  trackedNamespaces: state => {
    const seen = Object.create(null)
    state.tracked.images.forEach(image => {
      // counts[image.provider] = counts[image.provider] ? counts[image.provider] + 1 : 1
      seen[image.namespace] = true
    })
    return Object.keys(seen).length
  },
  trackedRegistries: state => {
    const seen = Object.create(null)
    state.tracked.images.forEach(image => {
      seen[image.registry] = true
    })
    return Object.keys(seen).length
  },
  // ----- stats ----
  // this function is used to transform stats data
  // into a form that the mini bar chart wants to consume
  updateStats: state => {
    const data = []
    var arrayLength = state.stats.stats.length
    const stats = state.stats.stats
    for (var i = 0; i < arrayLength; i++) {
      data.push({
        x: stats[i].date,
        y: stats[i].updates
      })
    }
    return data
  },
  approvalStats: state => {
    const data = []
    var arrayLength = state.stats.stats.length
    const stats = state.stats.stats
    for (var i = 0; i < arrayLength; i++) {
      data.push({
        x: stats[i].date,
        y: stats[i].approved
      })
    }
    return data
  },
  totalPods: state => {
    var arrayLength = state.resources.resources.length
    const resources = state.resources.resources
    let count = 0
    for (var i = 0; i < arrayLength; i++) {
      count += resources[i].status.replicas
    }
    return count
  },
  totalAvailablePods: state => {
    var arrayLength = state.resources.resources.length
    const resources = state.resources.resources
    let count = 0
    for (var i = 0; i < arrayLength; i++) {
      count += resources[i].status.availableReplicas
    }
    return count
  },
  totalUnavailablePods: state => {
    var arrayLength = state.resources.resources.length
    const resources = state.resources.resources
    let count = 0
    for (var i = 0; i < arrayLength; i++) {
      count += (resources[i].status.replicas - resources[i].status.availableReplicas)
    }
    return count
  }
}

export default getters
