<script setup lang="ts">
import { computed, h, nextTick, onMounted, reactive, ref } from 'vue';
import { onBeforeRouteUpdate, useRoute, useRouter } from 'vue-router';
import { NLayout, NLayoutHeader, NLayoutContent, NLayoutFooter, NButton, NBreadcrumb, NBreadcrumbItem, NSpace, NText, NSwitch, NAlert, NDropdown, NModal, NCard, useDialog, NProgress, useMessage } from 'naive-ui';
import backend, { Flag } from '@/api';
import type { DirItem, Effect, FileItem } from '@/api';

const route = useRoute();
const router = useRouter();

const dialog = useDialog();
const message = useMessage();

const path = ref((route.params.path || []) as string[]);

const files = ref<FileItem[]>([]);

const dirs = ref<DirItem[]>([]);

const active = ref(-1);

const showHidden = ref(false);

const contextMenu = reactive({
  show: false,
  x: 0,
  y: 0
});

const progressBar = reactive({
  show: false,
  text: "",
  progress: 0,
});

const fileOp = reactive<{
  from?: string[],
  move: boolean
}>({
  from: undefined,
  move: false
});

const sameOrigin = computed(() => {
  return path.value.length === fileOp.from!.length-1 && path.value.every((segment, index) => segment == fileOp.from?.[index]);
})

const properties = ref(false);

const nothing = computed(() => {
  return (
    files.value.filter(f => f.flag !== Flag.INVISIBLE || showHidden.value).length + 
    dirs.value.filter(f => f.flag !== Flag.INVISIBLE || showHidden.value).length <= 0
  );
});

const activeItem = computed(() => {
  if (active.value < 0) return undefined;
  return [...dirs.value, ...files.value][active.value]
})

const isFile = computed(() => {
  return active.value >= dirs.value.length;
})

const options = computed(() => {
  if (active.value < 0) return [];
  if (active.value < dirs.value.length) {
    return [
      {
        key: 'open',
        label: '打开文件夹',
        icon: () => h('i', { class: 'ri-folder-open-line' }),
      },
      {
        key: 'properties',
        label: '属性',
        icon: () => h('i', { class: 'ri-settings-line' })
      }
    ]
  }
  else {
    return [
      {
        key: 'open',
        label: '打开文件',
        icon: () => h('i', { class: 'ri-file-line' }),
        disabled: activeItem.value!.flag <= Flag.VISIBLE,
      },
      {
        key: 'download',
        label: '下载',
        icon: () => h('i', { class: 'ri-file-download-line' }),
        disabled: activeItem.value!.flag <= Flag.VISIBLE,
      },
      {
        key: 'copy',
        label: '复制',
        icon: () => h('i', { class: 'ri-file-copy-line' }),
        disabled: activeItem.value!.flag < Flag.READONLY
      },
      {
        key: 'move',
        label: '剪切',
        icon: () => h('i', { class: 'ri-scissors-2-line' }),
        disabled: activeItem.value!.flag < Flag.READWRITE
      },
      {
        key: 'replace',
        label: '替换',
        icon: () => h('i', { class: 'ri-loop-left-line' }),
        disabled: activeItem.value!.flag < Flag.READWRITE,
      },
      {
        key: 'properties',
        label: '属性',
        icon: () => h('i', { class: 'ri-settings-line' })
      }
    ]
  }
})

function onContextMenu(e: MouseEvent, index: number) {
  e.preventDefault();
  active.value = index;
  contextMenu.show = false;
  nextTick(() => {
    contextMenu.x = e.clientX;
    contextMenu.y = e.clientY;
    contextMenu.show = true;
  })
}

function onClickOutside() {
  contextMenu.show = false;
}

function openFile(name: string) {
  window.open(backend.fileUrl(...path.value, name));
}

async function selectFile(callback: (f: File) => void) {
  const input = document.createElement("input");
  input.type = "file";
  input.multiple = false;
  input.addEventListener("change", () => {
    if (input.files) {
      callback(input.files[0]);
    }
  });
  input.click();
}

function onSelect(key: any) {
  contextMenu.show = false;
  switch (key) {
    case 'open': {
      if (active.value < dirs.value.length) {
        forward(dirs.value[active.value].name);
      } else {
        window.open(backend.fileUrl(...path.value, files.value[active.value-dirs.value.length].name));
      }
      break;
    }
    case 'download': {
      window.open(backend.downloadUrl(...path.value, files.value[active.value-dirs.value.length].name));
      break;
    }
    case 'properties': {
      properties.value = true;
      break;
    }
    case 'replace': {
      selectFile(file => {
        dialog.warning({
          title: '覆盖',
          content: `使用本地的 ${file.name} 覆盖 ${activeItem.value!.name}？`,
          positiveText: '确定',
          negativeText: '取消',
          negativeButtonProps: { ghost: false, tertiary: true, type: 'error' },
          closable: false,
          maskClosable: false,
          closeOnEsc: false,
          transformOrigin: 'center',
          onPositiveClick() {
            backend.upload(file, (index, total) => {
              if (index < 0) {
                progressBar.show = true;
                progressBar.text = `${file.name} -> ${activeItem.value!.name}`;
                return;
              }
              progressBar.progress = index / total * 100;
            }, ...path.value, activeItem.value!.name)
            .then(() => {
              progressBar.show = false;
              update();
            })
          }
        })
      });
      break;
    }
    case 'copy': {
      fileOp.from = [...path.value, activeItem.value!.name];
      fileOp.move = false;
      break;
    }
    case 'move': {
      fileOp.from = [...path.value, activeItem.value!.name];
      fileOp.move = true;
      break;
    }
  }
}

async function update() {
  try {
    const result = await backend.tree(...path.value);
    files.value = result.files.sort((a, b) => a.name.toLowerCase().localeCompare(b.name.toLowerCase()));
    dirs.value = result.dirs.sort((a, b) => a.name.toLowerCase().localeCompare(b.name.toLowerCase()));
  } catch (e) {
    if (e === 500) {
      message.error('服务器内部错误。');
    }
  }
}

onMounted(async () => {
  await update();
})

onBeforeRouteUpdate(async to => {
  path.value = (to.params.path || []) as string[];
  active.value = -1;
  await update();
})

function forward(segment: string) {
  router.push('/' + [...path.value, segment].join('/'));
}

function to(path: string[]) {
  router.push('/' + path.join('/'));
}

function formatSize(n: number) {
  if (n < 1024) return `${n} B`;
  if (n < 1024 * 1024) return `${Math.round(n / 1024)} KB`
  if (n < 1024 * 1024 * 1024) return `${Math.round(n / 1024 / 1024)} MB`
  return `${Math.round(n / 1024 / 1024 / 1024)} GB`
}

function colorFromFlag(flag: Flag) {
  switch (flag) {
    case Flag.INVISIBLE: {
      return 'error';
    }
    case Flag.VISIBLE: {
      return 'warning';
    }
    case Flag.READWRITE: {
      return 'success';
    }
  }
}

function helpFromFlag(flag: Flag) {
  switch (flag) {
    case Flag.INVISIBLE: {
      return '+H'
    }
    case Flag.VISIBLE: {
      return '-R'
    }
    case Flag.READWRITE: {
      return '+W'
    }
  }
}

function textFromFlag(flag: Flag) {
  switch (flag) {
    case Flag.INVISIBLE: {
      return "invisible";
    }
    case Flag.VISIBLE: {
      return "visible";
    }
    case Flag.READONLY: {
      return "readonly";
    }
    case Flag.READWRITE: {
      return "readwrite";
    }
  }
}

function formatEffect(effect: Effect | null) {
  if (effect === null) {
    return '默认';
  }
  if (effect.direct) {
    return `定义在 \\${effect.definition} 中`;
  }
  else {
    return `继承自 \\${effect.cause}，定义在 \\${effect.definition} 中`;
  }
}

function upload() {
  selectFile(file => {
    dialog.warning({
      title: '上传',
      content: `上传 ${file.name}？`,
      positiveText: '确定',
      negativeText: '取消',
      negativeButtonProps: { ghost: false, tertiary: true, type: 'error' },
      closable: false,
      maskClosable: false,
      closeOnEsc: false,
      transformOrigin: 'center',
      onPositiveClick() {
        backend.upload(file, (index, total) => {
          if (index < 0) {
            progressBar.show = true;
            progressBar.text = `${file.name} -> ${file.name}`;
            return;
          }
          progressBar.progress = Math.round(index / total * 1000) / 10;
        }, ...path.value, file.name)
        .then(() => {
          progressBar.show = false;
          update();
        })
        .catch(code => {
          if (code === undefined) {
            message.error('由于目标文件访问控制，无法上传。');
          } else if (code === 1011 /* internal server error */) {
            message.error('服务器内部错误。');
          }
        });
      }
    })
  })
}

async function paste() {
  try {
    if (overlapsFile.value) {
      await new Promise<void>((resolve, reject) => {
        dialog.warning({
          title: '覆盖',
          content: `确定要覆盖 ${fileOp.from!.slice(-1)[0]} 吗？`,
          positiveText: '确定',
          negativeText: '取消',
          negativeButtonProps: { ghost: false, tertiary: true, type: 'error' },
          closable: false,
          maskClosable: false,
          closeOnEsc: false,
          transformOrigin: 'center',
          onPositiveClick() {
            resolve();
          },
          onNegativeClick() {
            reject();
          }
        })
      })
    }
    if (fileOp.move) {
      await backend.move(fileOp.from!, [...path.value, fileOp.from!.slice(-1)[0]]);
    }
    else {
      await backend.copy(fileOp.from!, [...path.value, fileOp.from!.slice(-1)[0]]);
    }
    fileOp.from = undefined;
  }
  catch (e) {
    if (e === 403) {
      message.error('由于目标文件访问控制，无法进行该操作。');
    }
    else if (e === 500){
      message.error('服务器内部错误。');
    }
  }
  finally {
    active.value = -1;
    await update();
  }
}

async function remove() {
  try {
    await new Promise<void>((resolve, reject) => {
      dialog.warning({
        title: '删除',
        content: `确定要删除 ${activeItem.value!.name} 吗？`,
        positiveText: '确定',
        negativeText: '取消',
        negativeButtonProps: { ghost: false, tertiary: true, type: 'error' },
        closable: false,
        maskClosable: false,
        closeOnEsc: false,
        transformOrigin: 'center',
        onPositiveClick() {
          resolve();
        },
        onNegativeClick() {
          reject();
        }
      })
    })
    await backend.delete(...path.value, activeItem.value!.name);
  }
  catch (e) {
    if (e === 500) {
      message.error('服务器内部错误。');
    }
  }
  finally {
    active.value = -1;
    await update();
  }
}

const overlapsFolder = computed(() => {
  return dirs.value.findIndex(d => d.name == fileOp.from!.slice(-1)[0]) >= 0;
});

const overlapsFile = computed(() => {
  return files.value.findIndex(f => f.name == fileOp.from!.slice(-1)[0]) >= 0;
});
</script>

<template>
  <div class="container">
    <NLayout>
      <NLayoutHeader>
        <div class="header">
          <div>
            <NButton quaternary size="small" @click="router.back">
              <i class="ri ri-arrow-left-line"></i>
            </NButton>
            <NButton quaternary size="small" @click="router.forward">
              <i class="ri ri-arrow-right-line"></i>
            </NButton>
            <NButton quaternary size="small" @click="to(path.slice(0, -1))" :disabled="!path.length">
              <i class="ri ri-arrow-up-line"></i>
            </NButton>
            <NButton quaternary size="small" @click="update">
              <i class="ri ri-refresh-line"></i>
            </NButton>
          </div>
          <NBreadcrumb separator="&gt;">
            <NBreadcrumbItem @click="to([])">
              <div class="breadcrumb-item-wrapper">
                <i class="ri-home-2-fill"></i>
              </div>
            </NBreadcrumbItem>
            <NBreadcrumbItem v-for="segment, index in path" @click="to(path.slice(0, index+1))">
              <div class="breadcrumb-item-wrapper">
                <i class="ri-folder-6-fill"></i>
                {{ segment }}
              </div>
            </NBreadcrumbItem>
          </NBreadcrumb>
          <div class="fileop-indicator" v-if="fileOp.from">
            正在{{ fileOp.move ? "移动" : "复制" }}
            {{ fileOp.from.join('\\') }}
          </div>
        </div>
      </NLayoutHeader>
      <NLayoutContent>
        <NSpace :wrap-item="false" style="height: 100%">
          <div class="toolbar">
            <NSpace vertical>
              <NButton quaternary @click="onSelect('move')" :disabled="!isFile || activeItem!.flag < Flag.READWRITE">
                <i class="ri-scissors-2-line"></i>
              </NButton>
              <NButton quaternary @click="onSelect('copy')" :disabled="!isFile || activeItem!.flag < Flag.READONLY">
                <i class="ri-file-copy-line"></i>
              </NButton>
              <NButton quaternary :disabled="!fileOp.from || sameOrigin || overlapsFolder" @click="paste">
                <i class="ri-clipboard-line"></i>
              </NButton>
              <NButton quaternary :disabled="!isFile || activeItem!.flag < Flag.READWRITE" @click="remove">
                <i class="ri-delete-bin-6-line"></i>
              </NButton>
              <NButton quaternary @click="upload">
                <i class="ri-upload-2-line"></i>
              </NButton>
              <div style="text-align: center;">
                <b><small>{{ showHidden ? "-H" : "+H" }}</small></b><br/>
                <NSwitch v-model:value="showHidden" :round="false"/>
              </div>
            </NSpace>
          </div>
          <div class="explorer">
            <div class="item">
              <div class="key name">
                名称
                <i class="ri-arrow-up-s-line"></i>
              </div>
              <div class="key">
                修改日期
              </div>
              <div class="key">
                类型
              </div>
              <div class="key size">
                大小
              </div>
            </div>
            <NSpace vertical style="row-gap: 2px">
              <div v-for="dir, index in dirs" :class="active === index ? 'item data focus' : 'item data'" 
                @click="active = index"
                @dblclick="forward(dir.name)"
                @contextmenu="onContextMenu($event, index)"
                :key="dir.name"
                v-show="dir.flag !== Flag.INVISIBLE || !showHidden">
                <div class="name">
                  <img :src="backend.dirIconSrc" alt="folder icon" height="20"/>
                  <div>
                    <NText :type="colorFromFlag(dir.flag)">
                      {{ dir.name }}
                      <b><small>{{ helpFromFlag(dir.flag) }}</small></b>
                    </NText>
                  </div>
                </div>
                <div class="key">
                  {{ dir.time.toLocaleString() }}
                </div>
                <div class="key">
                  文件夹
                </div>
              </div>
              <div v-for="file, index in files" :class="active === index + dirs.length ? 'item data focus' : 'item data'" 
                @click="active = index + dirs.length"
                @dblclick="file.flag >= Flag.READONLY && openFile(file.name)"
                @contextmenu="onContextMenu($event, index + dirs.length)"
                :key="file.name" 
                v-show="file.flag !== Flag.INVISIBLE || showHidden">
                <div class="key name">
                  <img :src="backend.iconSrc(...path, file.name)" alt="file icon" height="20"/>
                  <div>
                    <NText :type="colorFromFlag(file.flag)">
                      {{ file.name }}
                      <b><small>{{ helpFromFlag(file.flag) }}</small></b>
                    </NText> 
                  </div>
                </div>
                <div class="key">
                  {{ file.time.toLocaleString() }}
                </div>
                <div class="key">
                  {{ file.assoc }}
                </div>
                <div class="size">
                  {{ formatSize(file.size) }}
                </div>
              </div>
              <NAlert v-if='nothing' :bordered="false" type="info">
                文件夹为空。
              </NAlert>
            </NSpace>
          </div>
        </NSpace>
      </NLayoutContent>
      <NLayoutFooter>
        <div class="footer">
          2024 Github/origamizyt,
          Explorer Powered by Vue and NaiveUI
        </div>
      </NLayoutFooter>
    </NLayout>
  </div>
  <NDropdown 
    placement="bottom-start"
    trigger="manual"
    :show="contextMenu.show" 
    :x="contextMenu.x" 
    :y="contextMenu.y"
    :options="options"
    @clickoutside="onClickOutside"
    @select="onSelect" />
  <NModal v-model:show="properties" transform-origin="center">
    <NCard :bordered="false" size="huge" title="属性" style="width: 500px">
      <template #header-extra>
        <NButton quaternary type="error" @click="properties = false">
          <i class="ri-close-large-line"></i>
        </NButton>
      </template>
      <NSpace vertical>
        <NText>
          名称：{{ activeItem?.name }}
        </NText>
        <NText v-if="isFile">
          大小：{{ formatSize((activeItem as FileItem).size) }}
        </NText>
        <NText>
          修改日期：{{ activeItem?.time.toLocaleString() }}
        </NText>
        <NText v-if="isFile">
          类型：{{ (activeItem as FileItem).assoc.toLocaleString() }}
        </NText>
        <NText v-else>
          类型：文件夹
        </NText>
        <NText v-if="activeItem">
          访问级别：<NText tag="span" :type="colorFromFlag(activeItem.flag)" style="text-transform: uppercase;">
            <b>{{ textFromFlag(activeItem.flag) }}</b>
          </NText> ({{ formatEffect(activeItem.effect) }})
        </NText>
      </NSpace>
    </NCard>
  </NModal>
  <NModal v-model:show="progressBar.show" transform-origin="center" :close-on-esc="false" :mask-closable="false">
    <NSpace vertical class="progress-content">
      <NText>
        <b>
          {{ progressBar.text }}
        </b>
      </NText>
      <NProgress type="line" :percentage="progressBar.progress" processing>
        {{ progressBar.progress.toFixed(1) }} %
      </NProgress>
    </NSpace>
  </NModal>
</template>

<style scoped>
:deep(.progress-content) {
  min-width: 400px; 
  text-align: center; 
  user-select: none;
}

.header {
  padding: 10px;
  display: flex;
  gap: 15px;
}

.header .fileop-indicator {
  color: #909399;
}

.header * {
  align-self: center;
}

.header .n-breadcrumb {
  flex-grow: 1;
}

.breadcrumb-item-wrapper {
  padding: 0 3px;
}

:deep(.n-layout-scroll-container) {
  display: flex;
  flex-direction: column;
}

:deep(.toolbar) {
  padding: 5px;
  border-right: 1px solid #ffffff17;
}

:deep(.explorer) {
  flex-grow: 1;
  padding-right: 10px;
  height: 100%;
  overflow-y: hidden;
}

:deep(.explorer > .n-space) {
  height: calc(100% - 25px);
  overflow-y: scroll;
  scrollbar-width: none;
}

.item {
  padding: 2px 10px;
  display: flex;
  gap: 5px;
  user-select: none;
}

.item > .key {
  width: 20%;
}

.item.data > .size {
  text-align: right;
}

.item > .size {
  width: 10%;
}

.item > .name {
  display: flex;
  gap: 5px;
  width: 48%;
}

.item > *, .item > .name > * {
  align-self: center;
}

.item.data:hover, .item:not(.data) > .key:hover {
  background-color: rgba(255, 255, 255, .05);
}

.item.data.focus {
  background-color: rgba(255, 255, 255, .2);
}

.item:not(.data) > .key:not(:last-child) {
  border-right: 1px solid #ffffff17;
}

.footer {
  padding: 10px 0;
  text-align: center;
  color: #909399;
  font-size: 12px;
}
</style>