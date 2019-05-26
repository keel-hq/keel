<template>
  <div  class="page-header-index-wide">
    <a-card :bordered="false">
      <a-row>
        <a-col :sm="8" :xs="24">
          <head-info title="Last Event" :content="getLastEvent()" :bordered="true"/>
        </a-col>
        <a-col :sm="8" :xs="24">
          <head-info title="Audit Entries" :content="logs.length.toString()" :bordered="true"/>
        </a-col>
        <a-col :sm="8" :xs="24">
          <head-info title="Registries" content="-"/>
        </a-col>
      </a-row>
    </a-card>

    <a-card
      style="margin-top: 24px"
      :bordered="false"
      title="Audit Logs">

      <!-- table -->
      <a-table 
        :columns="columns"
        :dataSource="logs"
        :pagination="true"
        :loading="loading"
        
        :rowKey="log => log.id"
        size="middle">        
        <span slot="created" slot-scope="text, log">          
          {{ log.createdAt | time }}
        </span>       
        <span slot="metadata" slot-scope="text, log">
          <a-tag v-for="(item, key, index) in log.metadata" color="blue" :key="index">            
            {{key}}: {{item}}
          </a-tag>
          
        </span>       
      </a-table>
    </a-card>
  </div>
</template>

<script>
import HeadInfo from '@/components/tools/HeadInfo'
import { mapActions } from 'vuex'

export default {
  name: 'AuditLogs',
  components: {
    HeadInfo
  },
  data () {
    return {
      columns: [{
        title: 'Time',
        dataIndex: 'createdAt',
        key: 'createdAt',
        scopedSlots: { customRender: 'created' },
      }, {
        title: 'Action',
        dataIndex: 'action',
        key: 'action',
      }, {
        title: 'Resource Kind',
        dataIndex: 'resourceKind',
        key: 'resourceKind',
      }, {
        title: 'Identifier',
        dataIndex: 'identifier',
        key: 'identifier',
      }, {
        title: 'Metadata',
        key: 'metadata',
        dataIndex: 'metadata',
        width: 400,
        scopedSlots: { customRender: 'metadata' },
      }],
      loading: false,
      logs: [],
      pagination: {},
      tablePagination: {
        current: 1
      },
    }
  },

  watch: {
    '$store.state.audit.audit_logs' (logs) {
      this.logs = logs
    },   
    '$store.state.audit.pagination' (pagination) {
      this.pagination = pagination
    },
    '$store.state.audit.loading' (loading) {
      this.loading = loading
    }   
  },

  activated () {
    this.getAuditLogs()
  },

  methods: {
    handleTableChange (pagination, filters, sorter) {
      console.log('handle table change')
      console.log(pagination)

      console.log(filters)
      console.log(sorter)
    },

    getAuditLogs (pagination) {
      const query = {
        filter: '*',
        limit: 0,
        offset: 0
      }
      this.$store.dispatch('GetAuditLogs', query)
    },
    
    getLastEvent () {
      if (this.logs.length > 0) {
          let timestamp = this.logs[0].createdAt
          return new Date(timestamp).toLocaleDateString(undefined, {
            day: 'numeric',
            month: 'long',
            year: 'numeric',
            hour: '2-digit',
            minute: '2-digit',
            second: '2-digit'
          })
      }
      return '-'
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
