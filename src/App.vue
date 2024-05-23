<script setup lang="ts">
import { RouterView } from 'vue-router'
import { NConfigProvider, NDialogProvider, NMessageProvider, darkTheme } from 'naive-ui'
</script>

<template>
  <NConfigProvider :theme="darkTheme">
    <NDialogProvider>
      <NMessageProvider placement="top">
        <RouterView/>
      </NMessageProvider>
    </NDialogProvider>
  </NConfigProvider>
</template>