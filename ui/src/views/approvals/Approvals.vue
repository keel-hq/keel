<template>
  <div class="page-header-index-wide">
    <a-card :bordered="false">
      <a-row>
        <a-col :sm="8" :xs="24">
          <head-info title="Pending" :content="$store.getters.approvalsPending.length.toString()" :bordered="true"/>
        </a-col>
        <a-col :sm="8" :xs="24">
          <head-info title="Approved" :content="$store.getters.approvalsApprovedCount.toString()" :bordered="true"/>
        </a-col>
        <a-col :sm="8" :xs="24">
          <head-info title="Rejected" :content="$store.getters.approvalsRejectedCount.toString()"/>
        </a-col>
      </a-row>
    </a-card>

    <a-card
      style="margin-top: 24px"
      :bordered="false"
      title="Approvals">

      <div slot="extra">
        <a-radio-group>
          <a-radio-button @click="refresh()">Refresh</a-radio-button>
        </a-radio-group>
        <a-button type="primary" :ghost="true" @click="bulkApprove()" :disabled="!hasSelected" style="margin-left: 16px;">
          Approve
        </a-button>
        <a-button type="danger" :ghost="true" @click="bulkReject()" :disabled="!hasSelected" style="margin-left: 16px;">
          Reject
        </a-button>
        <a-input-search @search="onSearch" @change="onSearchChange" style="margin-left: 16px; width: 272px;" />
      </div>

      <!-- table -->
      <a-table
        :columns="columns"
        :dataSource="filtered()"
        :rowKey="approval => approval.id"
        :rowSelection="rowSelection"
        size="middle">
        >
        <span slot="updated" slot-scope="text, log">
          {{ log.updatedAt | time }}
        </span>
        <span slot="delta" slot-scope="text, approval">
          {{ approval.currentVersion }} -> {{ approval.newVersion }}
        </span>
        <span slot="votes" slot-scope="text, approval">
          {{ approval.votesReceived }}/{{ approval.votesRequired }}

        </span>
        <span slot="status" slot-scope="text, approval">
          <span v-if="approval.archived">
            Archived
            <a-progress :percent="100" :showInfo="false" />
          </span>
          <span v-else-if="approval.rejected">
            Rejected
            <a-progress :percent="100" :showInfo="false" status="exception" />
          </span>
          <span v-else-if="approval.votesReceived == approval.votesRequired">Complete
            <a-progress :percent="100" :showInfo="false"/>
          </span>
          <span v-else>Collecting...
            <a-progress :percent="getProgress(approval)" :showInfo="false" status="active" />
          </span>
        </span>
        <!-- deadline countdown -->
        <span slot="deadline" slot-scope="text, approval">
          <span v-if="isComplete(approval)">-</span>
          <a-tooltip v-else :title="approval.deadline" slot="action">
            <count-down :target="deadline(approval)" />
          </a-tooltip>
        </span>
        <span slot="action" slot-scope="text, approval">
          <a-button
            size="small"
            type="primary"
            icon="like"
            :disabled="isComplete(approval)"
            :loading="approval._loading"
            @click="approve(approval)"
          >
          </a-button>
          <a-divider type="vertical" />
          <!-- reject -->
          <a-button
            size="small"
            type="danger"
            icon="dislike"
            :disabled="isComplete(approval)"
            :loading="approval._loading"
            @click="reject(approval)"
          >
          </a-button>
          <a-divider type="vertical" />
          <!-- archive -->
          <a-divider type="vertical" />
          <a-tooltip title="Archive approval" slot="action">
            <a-button
              size="small"
              type="primary"
              icon="database"
              :disabled="approval.archived"
              :loading="approval._loading"
              @click="archive(approval)"
            >
            </a-button>
          </a-tooltip>
          <a-divider type="vertical" />
          <!-- delete approval -->
          <a-tooltip title="Delete approval request" slot="action">
            <a-button
              size="small"
              type="primary"
              icon="delete"
              :loading="approval._loading"
              @click="remove(approval)"
            >
            </a-button>
          </a-tooltip>
        </span>
      </a-table>
    </a-card>
  </div>
</template>

<script>
import HeadInfo from '@/components/tools/HeadInfo'
import CountDown from '@/components/CountDown'

export default {
  name: 'ApprovalsList',
  components: {
    HeadInfo,
    CountDown
  },
  data () {
    return {
      selectedRowKeys: [],
      selectedRows: [],
      rowSelection: {
        onChange: (selectedRowKeys, selectedRows) => {
          this.selectedRowKeys = selectedRowKeys
          this.selectedRows = selectedRows
        },
        getCheckboxProps: record => ({
          props: {
            disabled: record.rejected || record.archived,
            name: record.id
          }
        })
      },
      columns: [
        {
          dataIndex: 'updated',
          key: 'updated',
          title: 'Last Activity',
          scopedSlots: { customRender: 'updated' }
        }, {
          dataIndex: 'provider',
          key: 'provider',
          title: 'Provider'
        }, {
          title: 'Identifier',
          dataIndex: 'identifier',
          key: 'identifier'
        }, {
          title: 'Votes',
          dataIndex: 'votes',
          key: 'votes',
          scopedSlots: { customRender: 'votes' }
        }, {
          title: 'Delta',
          key: 'delta',
          dataIndex: 'delta',
          scopedSlots: { customRender: 'delta' }
        }, {
          title: 'Status',
          key: 'status',
          dataIndex: 'status',
          width: 200,
          scopedSlots: { customRender: 'status' }
        }, {
          title: 'Expires In',
          key: 'deadline',
          dataIndex: 'deadline',
          scopedSlots: { customRender: 'deadline' }
        }, {
          title: 'Action',
          key: 'action',
          scopedSlots: { customRender: 'action' }
        }],
      approvals: [],
      filter: ''
    }
  },

  watch: {
    '$store.state.approvals.approvals' (approvals) {
      this.approvals = approvals
    }
  },

  activated () {
    this.$store.dispatch('GetApprovals')
  },
  computed: {
    hasSelected () {
      return this.selectedRowKeys.length > 0
    }
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
        return this.approvals
      }
      const filter = this.filter
      return this.approvals.reduce(function (filtered, approval) {
        if (approval.identifier.includes(filter)) {
          filtered.push(approval)
          return filtered
        } else if (approval.provider.includes(filter)) {
          filtered.push(approval)
          return filtered
        } else if (approval.message.includes(filter)) {
          filtered.push(approval)
          return filtered
        } else if (approval.createdAt.includes(filter)) {
          filtered.push(approval)
          return filtered
        }
        return filtered
      }, [])
    },

    refresh () {
      this.$store.dispatch('GetApprovals')
      this.$notification.info({
        message: 'Updating..',
        description: `fetching approvals`
      })
    },

    isComplete (approval) {
      return (approval.archived || approval.rejected || approval.votesReceived >= approval.votesRequired)
    },

    deadline (approval) {
      return new Date(approval.deadline)
    },

    getProgress (approval) {
      if (approval.votesReceived === 0) { return 0 }
      return (approval.votesReceived * 100) / approval.votesRequired
    },

    approve (approval) {
      const that = this
      this.$confirm({
        title: 'Confirm update',
        content: `are you sure want to approve update for ${approval.identifier}?`,
        onOk () {
          that.updateApproval(approval, 'approve')
        },
        onCancel () {
        }
      })
    },

    remove (approval) {
      const that = this
      this.$confirm({
        title: 'Confirm deletion',
        content: `are you sure want to delete approval ${approval.identifier}?`,
        onOk () {
          that.updateApproval(approval, 'delete')
        },
        onCancel () {
        }
      })
    },

    archive (approval) {
      const that = this
      this.$confirm({
        title: 'Confirm archive',
        content: `are you sure want to archive approval ${approval.identifier}?`,
        onOk () {
          that.updateApproval(approval, 'archive')
        },
        onCancel () {
        }
      })
    },

    bulkApprove () {
      const that = this
      this.$confirm({
        title: 'Confirm update',
        content: `Are you sure want to approve the selected updates?`,
        onOk () {
          for (let i = 0; i < that.selectedRows.length; i++) {
            if (!that.selectedRows[i].rejected && !that.selectedRows[i].archived) {
              that.updateApproval(that.selectedRows[i], 'approve')
            }
          }
        },
        onCancel () {
        }
      })
    },
    bulkReject () {
      const that = this
      this.$confirm({
        title: 'Confirm update',
        content: `Are you sure want to reject the selected updates?`,
        onOk () {
          for (let i = 0; i < that.selectedRows.length; i++) {
            if (!that.selectedRows[i].rejected && !that.selectedRows[i].archived) {
              that.updateApproval(that.selectedRows[i], 'reject')
            }
          }
        },
        onCancel () {
        }
      })
    },

    updateApproval (approval, action) {
      const payload = {
        id: approval.id,
        identifier: approval.identifier,
        action: action,
        voter: 'admin-web-ui'
      }

      let msg = ''
      let desc = ''

      switch (action) {
        case 'approve':
          msg = 'Approved!'
          desc = `${approval.identifier} approved successfuly!`
          break
        case 'reject':
          msg = 'Updated rejected!'
          desc = `${approval.identifier} update rejected!`
          break
        case 'delete':
          msg = 'Approval deleted!'
          desc = `${approval.identifier} approval deleted!`
          break
        case 'archive':
          msg = 'Approval entry archived!'
          desc = `${approval.identifier} approval archived!`
          break
        default:
          break
      }

      this.$store.dispatch('UpdateApproval', payload).then(() => {
        const error = this.$store.state.resources.error
        if (error === null) {
          this.$notification.success({
            message: msg,
            description: desc
          })
        } else {
          this.$notification['error']({
            message: `${action} error`,
            description: `Error: ${error.body}`,
            duration: 4
          })
        }
        this.$store.dispatch('GetApprovals')
      })
    },

    reject (approval) {
      this.updateApproval(approval, 'reject')
    }
  }
}
</script>

<style lang="less" scoped>
    .ant-avatar-lg {
        width: 48px;
        height: 48px;
        line-height: 48px;
    }

    .list-content-item {
        color: rgba(0, 0, 0, .45);
        display: inline-block;
        vertical-align: middle;
        font-size: 14px;
        margin-left: 40px;
        span {
            line-height: 20px;
        }
        p {
            margin-top: 4px;
            margin-bottom: 0;
            line-height: 22px;
        }
    }
</style>
