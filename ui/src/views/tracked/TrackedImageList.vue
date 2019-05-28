<template>
  <div class="page-header-index-wide">
    <a-card :bordered="false">
      <a-row>
        <a-col :sm="8" :xs="24">
          <head-info title="Namespaces" :content="$store.getters.trackedNamespaces.toString()" :bordered="true"/>
        </a-col>
        <a-col :sm="8" :xs="24">
          <head-info title="Total Images Tracked" :content="images.length.toString()" :bordered="true"/>
        </a-col>
        <a-col :sm="8" :xs="24">
          <head-info title="Registries" :content="$store.getters.trackedRegistries.toString()"/>
        </a-col>
      </a-row>
    </a-card>

    <a-card
      style="margin-top: 24px"
      :bordered="false"
      title="Tracked Images"
    >
      <div slot="extra">
        <a-radio-group>
          <a-radio-button @click="refresh()">Refresh</a-radio-button>
        </a-radio-group>
        <a-input-search @search="onSearch" @change="onSearchChange" style="margin-left: 16px; width: 272px;" />
      </div>
      <!-- table -->
      <a-table
        :columns="columns"
        :dataSource="filtered()"
        :rowKey="image => image.id"
        size="middle"
      >
        <span slot="trigger" slot-scope="text, image">
          <span v-if="image.trigger == 'poll'">poll - {{ image.pollSchedule }}</span>
          <span v-else>webhook/GCR</span>
        </span>
      </a-table>
    </a-card>
  </div>
</template>

<script>
import HeadInfo from '@/components/tools/HeadInfo'

export default {
  name: 'TrackedImageList',
  components: {
    HeadInfo
  },
  data () {
    return {
      columns: [{
        dataIndex: 'image',
        key: 'image',
        title: 'Image Name'
      }, {
        title: 'Provider',
        dataIndex: 'provider',
        key: 'provider'
      }, {
        title: 'Namespace',
        dataIndex: 'namespace',
        key: 'namespace'
      }, {
        title: 'Policy',
        dataIndex: 'policy',
        key: 'policy'
      }, {
        title: 'Trigger',
        key: 'trigger',
        dataIndex: 'trigger',
        scopedSlots: { customRender: 'trigger' }
      }],
      images: [],
      filter: ''
    }
  },

  watch: {
    '$store.state.tracked.images' (images) {
      this.images = images
    }
  },

  activated () {
    this.$store.dispatch('GetTrackedImages')
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
        return this.images
      }
      const filter = this.filter
      return this.images.reduce(function (filtered, image) {
        if (image.image.includes(filter)) {
          filtered.push(image)
          return filtered
        } else if (image.namespace.includes(filter)) {
          filtered.push(image)
          return filtered
        } else if (image.policy.includes(filter)) {
          filtered.push(image)
          return filtered
        } else if (image.trigger.includes(filter)) {
          filtered.push(image)
          return filtered
        } else if (image.pollSchedule.includes(filter)) {
          filtered.push(image)
          return filtered
        } else if (image.trigger.includes(filter)) {
          filtered.push(image)
          return filtered
        } else if (image.registry.includes(filter)) {
          filtered.push(image)
          return filtered
        }
        return filtered
      }, [])
    },

    refresh () {
      this.$store.dispatch('GetTrackedImages')
      this.$notification.info({
        message: 'Updating..',
        description: `fetching tracked images`
      })
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
