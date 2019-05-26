<template>
  <div class="antv-chart-mini">
    <div class="chart-wrapper" :style="{ height: 46 }">
      <v-chart :force-fit="true" :height="height" :data="data" :padding="[36, 5, 18, 5]">
        <v-tooltip />
        <v-bar position="x*y" />
      </v-chart>
    </div>
  </div>
</template>

<script>

const tooltip = [
  'x*y',
  (x, y) => ({
    name: x,
    value: y
  })
]

const scale = [{
  dataKey: 'x',
  min: 2
}, {
  dataKey: 'y',
  title: 'Date',
  min: 1,
  max: 7
}]

export default {
  name: 'MiniBar',
  data () {
    return {      
      tooltip,
      scale,
      height: 100
    }
  },
  props: {
    data: Array
  }
}
</script>

<style lang="less" scoped>
  @import "chart";
</style>
