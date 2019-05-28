<template>
  <div class="page-header-index-wide">
    <a-row :gutter="24">
      <a-col :sm="24" :md="12" :xl="6" :style="{ marginBottom: '24px' }">
        <chart-card :loading="loading" title="Cluster Resources" :total="$store.getters.resourcesAll.length">
          <a-tooltip title="Resources observed by Kubernetes provider" slot="action">
            <a-icon type="info-circle-o" />
          </a-tooltip>

          <template slot="footer">Managed by Keel: <span>{{ $store.getters.resourcesManaged.length }}</span></template>
        </chart-card>
      </a-col>
      <a-col :sm="24" :md="12" :xl="6" :style="{ marginBottom: '24px' }">
        <chart-card :loading="loading" title="Total pods in cluster" :total="$store.getters.totalPods | NumberFormat">
          <trend :reverseColor="true" flag="up" style="margin-right: 16px;">
            <span slot="term">Healthy</span>
            {{ $store.getters.totalAvailablePods }}
          </trend>
          <trend :reverseColor="true" flag="down">
            <span slot="term">Unavailable</span>
            {{ $store.getters.totalUnavailablePods }}
          </trend>
          <template slot="footer">Percent up:<span> {{ percentUp() | round(2) }}%</span></template>
        </chart-card>
      </a-col>
      <a-col :sm="24" :md="12" :xl="6" :style="{ marginBottom: '24px' }">
        <chart-card :loading="loading" title="Updates" :total="$store.state.stats.totalUpdatesThisPeriod | NumberFormat">
          <a-tooltip title="Daily updates" slot="action">
            <a-icon type="info-circle-o" />
          </a-tooltip>
          <div>
            <mini-bar :data="$store.getters.updateStats"/>
          </div>
          <template slot="footer">Average <span>{{ $store.state.stats.totalUpdatesThisPeriod/4 | round(0,0) }}</span> updates per week</template>
        </chart-card>
      </a-col>
      <a-col :sm="24" :md="12" :xl="6" :style="{ marginBottom: '24px' }">
        <chart-card :loading="loading" title="Pending Approvals" :total="$store.getters.approvalsPending.length">
          <a-tooltip title="Updates in progress (awaiting approval)" slot="action">
            <a-icon type="info-circle-o" />
          </a-tooltip>
          <div>
            <mini-bar :data="$store.getters.approvalStats"/>
          </div>
          <template slot="footer">
            <trend :reverseColor="true" flag="up" style="margin-right: 16px;">
              <span slot="term">A</span>
              {{ $store.getters.approvalsApprovedCount }}
            </trend>
            <trend :reverseColor="true" flag="down">
              <span slot="term">R</span>
              {{ $store.getters.approvalsRejectedCount }}
            </trend>
          </template>
        </chart-card>
      </a-col>
    </a-row>

    <a-modal
      :title="`Set policy for ${resourceUnderPolicyChange.identifier}`"
      :visible="visible"
      @ok="handleOk()"
      :confirmLoading="confirmLoading"
      @cancel="handleCancel"
    >
      <span v-if="policyUnderChange === 'glob'">
        <div class="meta-content" slot="description">
          Use wildcards to match tags. Policy <strong>glob:build-*"</strong> would
          match tags such as <strong>build-1</strong>, <strong>build-2</strong>,
          <strong>build-commit-id-5</strong>:
        </div>
        <a-input addonBefore="glob:" placeholder="build-*" v-model="policyInput" />

      </span>

      <span v-if="policyUnderChange === 'regexp'">
        <div class="meta-content" slot="description">
          Use regular expressions to match versions, regexp syntax can be found
          here: <a href="https://github.com/google/re2/wiki/Syntax" target="_blank">https://github.com/google/re2/wiki/Syntax</a>.
        </div>
        <a-input addonBefore="regexp:" placeholder="^([a-zA-Z]+)$" v-model="policyInput" />
      </span>

    </a-modal>

    <!-- <a-row :gutter="24"> -->
    <a-card
      style="margin-top: 24px margin-bot: 24px"
      :bordered="false"
      title="Kubernetes Cluster Resources">

      <div slot="extra">
        <a-radio-group>
          <a-radio-button @click="refresh()">Refresh</a-radio-button>
        </a-radio-group>
        <a-input-search @search="onSearch" @change="onSearchChange" style="margin-left: 16px; width: 272px;" />
      </div>
      <!-- table -->
      <a-table
        :rowKey="resource => resource.identifier"
        :columns="columns"
        :dataSource="filtered()"

        size="middle">
        <!-- resource kind/name -->
        <span slot="name" slot-scope="text, resource">
          {{ resource.kind }}/{{ resource.name }}
        </span>
        <span slot="pods" slot-scope="text, resource">
          <a-tooltip placement="top" >
            <template slot="title">
              <span>Currently {{ resource.status.availableReplicas }} pods are available out of {{ resource.status.replicas }} desired</span>
            </template>
            <a-badge v-if="availabilityOK(resource)" status="success" :text="getAvailability(resource)"/>
            <a-badge v-else status="warning" :text="getAvailability(resource)"/>
          </a-tooltip>

        </span>
        <!-- update policies -->
        <span slot="policy" slot-scope="text, resource">
          <a-badge v-if="resource.policy === 'nil policy'" status="default" text="none" />
          <a-badge v-else status="success" :text="resource.policy"/>
        </span>
        <span slot="approvals" slot-scope="text, resource">
          {{ resource._required_approvals ? resource._required_approvals : '-' }}
        </span>
        <!-- labels -->
        <span slot="labels" slot-scope="text, resource">
          <a-tag v-for="(item, key, index) in resource._keel_opts" color="blue" :key="index">
            {{ key }}: {{ item }}
          </a-tag>
        </span>
        <span slot="images" slot-scope="text, resource">
          <a-tag v-for="(item, index) in resource.images" :key="index">
            {{ item }}
          </a-tag>
        </span>
        <!-- actions -->
        <span slot="action" slot-scope="text, resource">
          <a-button
            size="small"
            type="primary"
            icon="pause"
            :disabled="resource.policy === 'nil policy'"
            :loading="resource._loading"
            @click="setPolicy(resource, 'never')">
            Pause
          </a-button>
          &nbsp;
          <a-dropdown >
            <a-menu slot="overlay">
              <a-menu-item @click="setPolicy(resource, 'patch')" key="1">patch</a-menu-item>
              <a-menu-item @click="setPolicy(resource, 'minor')" key="2">minor</a-menu-item>
              <a-menu-item @click="setPolicy(resource, 'major')" key="3">major</a-menu-item>
              <a-menu-item @click="setPolicy(resource, 'all')" key="4">all</a-menu-item>
              <a-menu-item @click="setPolicy(resource, 'force')" key="5">force</a-menu-item>
              <a-menu-item @click="showPolicyModal(resource, 'glob')" key="6">glob</a-menu-item>
              <a-menu-item @click="showPolicyModal(resource, 'regexp')" key="7">regexp</a-menu-item>
            </a-menu>
            <a-button size="small" type="primary">
              Policy<a-icon type="down" />
            </a-button>
          </a-dropdown>
          &nbsp;
          <a-tooltip placement="top" >
            <a-button-group>
              <a-button size="small" type="primary" icon="up" @click="setApproval(resource, true)"></a-button>
              <a-button size="small" type="primary" icon="down" @click="setApproval(resource, false)"></a-button>
            </a-button-group>
          </a-tooltip>
          &nbsp;
          <a-tooltip placement="top" >
            <template slot="title">
              <span>Enable or disable active registry polling for the images (defaults to polling every minute)</span>
            </template>
            <!-- poll control -->
            <a-switch :checked="resource._trigger_poll" @click="toggleTracking(resource)" :disabled="resource.policy === 'nil policy'" >
              <a-icon type="sync" slot="checkedChildren"/>
              <a-icon type="disconnect" slot="unCheckedChildren"/>
            </a-switch>
          </a-tooltip>
        </span>
      </a-table>
    </a-card>

  </div>
</template>

<script>
import { ChartCard, MiniArea, MiniBar, MiniProgress, RankList, Bar, Trend, NumberInfo, MiniSmoothArea } from '@/components'
import { mixinDevice } from '@/utils/mixin'

export default {
  name: 'Analysis',
  mixins: [mixinDevice],
  components: {
    ChartCard,
    MiniArea,
    MiniBar,
    MiniProgress,
    RankList,
    Bar,
    Trend,
    NumberInfo,
    MiniSmoothArea
  },
  data () {
    return {
      timer: '',
      loading: true,
      filter: '',
      resources: [],

      columns: [{
        title: 'Namespace',
        dataIndex: 'namespace',
        key: 'namespace'
      }, {
        dataIndex: 'name',
        key: 'name',
        title: 'Name',
        scopedSlots: { customRender: 'name' }
      }, {
        dataIndex: 'pods',
        key: 'pods',
        title: 'Pods',
        scopedSlots: { customRender: 'pods' }
      }, {
        title: 'Policy',
        dataIndex: 'policy',
        // width: 120,
        key: 'policy',
        scopedSlots: { customRender: 'policy' }
      }, {
        title: 'Required Approvals',
        dataIndex: 'approvals',
        key: 'approvals',
        scopedSlots: { customRender: 'approvals' }
      }, {
        title: 'Images',
        key: 'images',
        dataIndex: 'images',
        width: 180,
        scopedSlots: { customRender: 'images' }
      }, {
        title: 'Keel Labels & Annotations',
        key: 'labels',
        dataIndex: 'labels',
        width: 230,
        scopedSlots: { customRender: 'labels' }
      }, {
        title: 'Policy & Approvals Control',
        key: 'action',
        // fixed: 'right',
        scopedSlots: { customRender: 'action' }
      }],

      // glob/regexp policy settings
      confirmLoading: false,
      visible: false,
      policyInput: '',
      policyUnderChange: '',
      resourceUnderPolicyChange: {}
    }
  },
  created () {
    setTimeout(() => {
      this.loading = !this.loading
    }, 500)
  },

  activated () {
    this.fetchData()
  },

  watch: {
    '$store.state.resources.resources' (resources) {
      this.resources = resources
    }
  },

  beforeDestroy () {
    clearInterval(this.timer)
  },

  methods: {
    onSearch (value) {
      this.filter = value
    },
    onSearchChange (e) {
      this.filter = e.target._value
    },

    filtered () {
      if (this.filter === '') {
        return this.resources
      }
      const filter = this.filter
      return this.resources.reduce(function (filtered, resource) {
        if (resource.identifier.includes(filter)) {
          filtered.push(resource)
          return filtered
        } else if (resource.namespace.includes(filter)) {
          filtered.push(resource)
          return filtered
        } else if (resource.policy.includes(filter)) {
          filtered.push(resource)
          return filtered
        } else if (resource.provider.includes(filter)) {
          filtered.push(resource)
          return filtered
        } else {
          // checking images
          var arrayLength = resource.images.length
          const images = resource.images
          for (var i = 0; i < arrayLength; i++) {
            if (images[i].includes(filter)) {
              filtered.push(resource)
              return filtered
            }
          }
        }
        return filtered
      }, [])
    },

    setPolicy (resource, policy) {
      const payload = {
        identifier: resource.identifier,
        policy: policy,
        provider: resource.provider
      }
      this.$store.dispatch('SetResourcePolicy', payload).then(() => {
        const error = this.$store.state.resources.error
        if (error === null) {
          this.$notification.success({
            message: 'Policy updated!',
            description: `${resource.kind} ${resource.name} policy set to ${policy}!`
          })
        } else {
          this.$notification['error']({
            message: 'Failed to update policy',
            description: `Error: ${error.body}`,
            duration: 4
          })
        }
        this.$store.dispatch('GetResources')
      })
    },

    toggleTracking (resource) {
      const payload = {
        identifier: resource.identifier,
        provider: resource.provider
      }

      if (!resource._trigger_poll) {
        payload.trigger = 'poll'
      } else {
        payload.trigger = 'default'
      }

      this.$store.dispatch('SetTracking', payload).then(() => {
        const error = this.$store.state.tracked.error
        if (error === null) {
          this.$notification.success({
            message: 'Image tracking updated!',
            description: `${resource.kind} ${resource.name} trigget set to ${payload.trigger}!`
          })
        } else {
          this.$notification['error']({
            message: 'Failed to update trigger',
            description: `Error: ${error.body}`,
            duration: 4
          })
        }
        this.$store.dispatch('GetResources')
      })
    },

    setApproval (resource, increase) {
      const payload = {
        identifier: encodeURI(resource.identifier),
        provider: resource.provider
      }

      const current = resource.annotations['keel.sh/approvals']
      if (increase) {
        // increasing approvals count
        if (current) {
          payload.votesRequired = parseInt(current, 10) + 1
        } else {
          payload.votesRequired = 1
        }
      } else {
        // decreasing approvals count
        if (current > 1) {
          payload.votesRequired = parseInt(current, 10) - 1
        } else {
          payload.votesRequired = 0
        }
      }

      this.$store.dispatch('SetApproval', payload).then(() => {
        const error = this.$store.state.approvals.error
        if (error === null) {
          this.$notification.success({
            message: 'Resource approvals updated!',
            description: `${resource.kind} ${resource.name} approvals set to ${payload.votesRequired}!`
          })
        } else {
          this.$notification['error']({
            message: 'Failed to resource approval',
            description: `Error: ${error.body}`,
            duration: 4
          })
        }
        this.$store.dispatch('GetResources')
      })
    },

    refresh () {
      this.fetchData()
      this.$notification.info({
        message: 'Updating..',
        description: `fetching approvals, resources and stats`
      })
    },

    fetchData () {
      this.$store.dispatch('GetApprovals')
      this.$store.dispatch('GetResources')
      this.$store.dispatch('GetStats')
    },

    startPolling () {
      this.timer = setInterval(this.fetchData, 2500)
    },

    stopPolling () {
      clearInterval(this.timer)
    },

    handleSetMenuClick (e) {
      console.log('click', e)
    },

    showPolicyModal (resource, policyName) {
      this.policyUnderChange = policyName
      this.resourceUnderPolicyChange = resource
      this.visible = true
    },

    handleOk (e) {
      console.log(e)
      // this.ModalText = 'The modal will be closed after two seconds';
      this.confirmLoading = true
      let policy = ''
      if (this.policyUnderChange === 'glob') {
        policy = 'glob:' + this.policyInput
      } else if (this.policyUnderChange === 'regexp') {
        policy = 'regexp:' + this.policyInput
      }
      this.visible = false
      this.confirmLoading = false
      this.setPolicy(this.resourceUnderPolicyChange, policy)
      // reset the form
      this.policyUnderChange = ''
      this.resourceUnderPolicyChange = {}
    },
    handleCancel (e) {
      console.log('Clicked cancel button')
      this.visible = false
      this.policyUnderChange = ''
      this.resourceUnderPolicyChange = {}
    },

    getAvailability (resource) {
      return `${resource.status.availableReplicas}/${resource.status.replicas}`
    },

    availabilityOK (resource) {
      return resource.status.availableReplicas === resource.status.replicas
    },

    percentUp () {
      if (this.$store.getters.totalPods === 0) {
        return 0
      }

      const p = 100 - (this.$store.getters.totalUnavailablePods * 100) / this.$store.getters.totalPods

      return p
    }
  }
}
</script>

<style lang="less" scoped>
  .extra-wrapper {
    line-height: 55px;
    padding-right: 24px;

    .extra-item {
      display: inline-block;
      margin-right: 24px;

      a {
        margin-left: 24px;
      }
    }
  }

  .antd-pro-pages-dashboard-analysis-twoColLayout {
    position: relative;
    display: flex;
    display: block;
    flex-flow: row wrap;

    &.desktop div[class^=ant-col]:last-child {
      position: absolute;
      right: 0;
      height: 100%;
    }
  }

  .antd-pro-pages-dashboard-analysis-salesCard {
    height: calc(100% - 24px);
    /deep/ .ant-card-head {
      position: relative;
    }
  }

  .dashboard-analysis-iconGroup {
    i {
      margin-left: 16px;
      color: rgba(0,0,0,.45);
      cursor: pointer;
      transition: color .32s;
      color: black;
    }
  }
  .analysis-salesTypeRadio {
    position: absolute;
    right: 54px;
    bottom: 12px;
  }
</style>
