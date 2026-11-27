<script setup>
import { computed } from 'vue'
import { inline } from '../markdown'
const props = defineProps({ text: { type: String, default: '' } })
const parts = computed(() => inline(props.text))
</script>
<template>
  <template v-for="(p, i) in parts" :key="i">
    <strong v-if="p.t === 'b'">{{ p.v }}</strong>
    <em v-else-if="p.t === 'i'">{{ p.v }}</em>
    <code v-else-if="p.t === 'code'">{{ p.v }}</code>
    <template v-else>{{ p.v }}</template>
  </template>
</template>
